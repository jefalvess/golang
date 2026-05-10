package handler

import (
	"comparify/internal/service"
	"comparify/pkg/customerror"
	"errors"
	"fmt"
	"net/http"

	"comparify/pkg/logger"
	"comparify/pkg/utils"

	"github.com/labstack/echo/v4"
)

// errorCode é um identificador legível por máquina para cada classe de erro da API.
type errorCode string

const (
	errCodeBadRequest    errorCode = "BAD_REQUEST"
	errCodeNotFound      errorCode = "NOT_FOUND"
	errCodeInternalError errorCode = "INTERNAL_ERROR"
)

// apiErrorResponse é o envelope de erro padronizado retornado pela API.
type apiErrorResponse struct {
	Code    errorCode `json:"code"`
	Message string    `json:"message"`
	Status  int       `json:"status"`
}

type Handler struct {
	productService service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{productService: svc}
}

func (h *Handler) ListItems(c echo.Context) error {
	const componentName = "Handler.ListItems"

	requestLogger := logger.Logger.With(
		"component", componentName,
		"method", c.Request().Method,
		"path", c.Path(),
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
	)

	items, err := h.productService.ListItems(c.Request().Context())
	if err != nil {
		var customErr *customerror.CustomError
		if !errors.As(err, &customErr) {
			requestLogger.Errorw("list items failed", "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}

		switch customErr.Msg {
		case customerror.InvalidRequestError:
			requestLogger.Warnw("list items failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusBadRequest, errCodeBadRequest, customErr.Msg)
		case customerror.NotFoundError:
			requestLogger.Warnw("list items not found", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusNotFound, errCodeNotFound, "produto(s) não encontrado(s)")
		case customerror.RequestExecutionError, customerror.InternalError:
			requestLogger.Errorw("list items failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		default:
			requestLogger.Errorw("list items failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"items":           items,
		"count":           len(items),
		"availableFields": h.productService.AvailableFields(),
	})
}

func (h *Handler) Compare(c echo.Context) error {
	const componentName = "Handler.Compare"

	requestLogger := logger.Logger.With(
		"component", componentName,
		"method", c.Request().Method,
		"path", c.Path(),
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
	)

	ids, err := parseCompareRequest(c)
	if err != nil {
		var customErr *customerror.CustomError
		if !errors.As(err, &customErr) {
			requestLogger.Errorw("compare failed", "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}

		switch customErr.Msg {
		case customerror.InvalidRequestError:
			requestLogger.Warnw("compare failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusBadRequest, errCodeBadRequest, customErr.Msg)
		case customerror.NotFoundError:
			requestLogger.Warnw("compare not found", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusNotFound, errCodeNotFound, "produto(s) não encontrado(s)")
		default:
			requestLogger.Errorw("compare failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}
	}

	compareLogger := requestLogger.With("ids", ids)
	items, err := h.productService.Compare(c.Request().Context(), ids, c.QueryParam("fields"))
	if err != nil {
		var customErr *customerror.CustomError
		if !errors.As(err, &customErr) {
			compareLogger.Errorw("compare failed", "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}

		switch customErr.Msg {
		case customerror.InvalidRequestError:
			compareLogger.Warnw("compare failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusBadRequest, errCodeBadRequest, customErr.Msg)
		case customerror.NotFoundError:
			compareLogger.Warnw("compare not found", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusNotFound, errCodeNotFound, "produto(s) não encontrado(s)")
		default:
			compareLogger.Errorw("compare failed", "error_code", customErr.Msg, "error", err)
			return writeErrorResponse(c, http.StatusInternalServerError, errCodeInternalError, "erro interno do servidor")
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

// parseCompareRequest lê e valida os parâmetros de query do endpoint /compare.
// Comentário: Retorna os ids solicitados ou um erro sentinela se os parâmetros forem inválidos.
func parseCompareRequest(c echo.Context) ([]string, error) {
	const componentName = "Handler.parseCompareRequest"

	// Valida se todos os parâmetros são permitidos
	for key := range c.QueryParams() {
		if !isAllowedCompareParam(key) {
			// Mensagem de erro em inglês para padrão internacional
			return nil, customerror.ThrowNew(
				componentName,
				customerror.InvalidRequestError,
				fmt.Errorf("%w: %q is not allowed", service.ErrInvalidQueryParam, key),
			)
		}
	}

	idsParam := c.QueryParam("ids")
	if idsParam == "" {
		// Mensagem de erro em inglês para padrão internacional
		return nil, customerror.ThrowNew(
			componentName,
			customerror.InvalidRequestError,
			fmt.Errorf("%w: ids query parameter is required", service.ErrInvalidQueryParam),
		)
	}

	ids := utils.SplitAndTrim(idsParam, ",")
	if len(ids) == 0 {
		// Mensagem de erro em inglês para padrão internacional
		return nil, customerror.ThrowNew(
			componentName,
			customerror.InvalidRequestError,
			fmt.Errorf("%w: must include at least one id", service.ErrInvalidQueryParam),
		)
	}

	return ids, nil
}

func isAllowedCompareParam(key string) bool {
	switch key {
	case "ids", "fields":
		return true
	default:
		return false
	}
}

func writeErrorResponse(c echo.Context, status int, code errorCode, message string) error {
	return c.JSON(status, map[string]any{
		"error": apiErrorResponse{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}
