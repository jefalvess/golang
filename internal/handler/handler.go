package handler

import (
	"comparify/internal/service"
	"errors"
	"net/http"

	"comparify/internal/repository"
	"comparify/pkg/utils"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	productService service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{productService: svc}
}

func (h *Handler) GetItem(c echo.Context) error {
	itemID := c.Param("id")
	if itemID == "" {
		return writeError(c, http.StatusNotFound, "item not found")
	}

	item, err := h.productService.GetItem(c.Request().Context(), itemID, c.QueryParam("fields"))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProductNotFound):
			return writeError(c, http.StatusNotFound, "item not found")
		case errors.Is(err, service.ErrInvalidFieldSelection):
			return writeError(c, http.StatusBadRequest, err.Error())
		default:
			return writeError(c, http.StatusInternalServerError, "failed to load item")
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"item": item,
	})
}

func (h *Handler) Compare(c echo.Context) error {
	ids, err := readCompareSelection(c)
	if err != nil {
		return writeError(c, http.StatusBadRequest, err.Error())
	}

	items, err := h.productService.Compare(c.Request().Context(), ids, c.QueryParam("fields"))
	if err != nil {
		if errors.Is(err, service.ErrInvalidFieldSelection) {
			return writeError(c, http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, repository.ErrProductNotFound) {
			return writeError(c, http.StatusNotFound, "items not found")
		}

		return writeError(c, http.StatusInternalServerError, "failed to load items")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

// readCompareSelection mantém o contrato do compare explícito: apenas ids conhecidos
// e projeção opcional de campos. Qualquer outro parâmetro é tratado como erro.
func readCompareSelection(c echo.Context) ([]string, error) {
	queryParams := c.QueryParams()
	for queryKey := range queryParams {
		if !isSupportedCompareQueryKey(queryKey) {
			return nil, errors.New("unsupported compare query parameter: " + queryKey)
		}
	}

	idsRaw := c.QueryParam("ids")
	ids := utils.SplitAndTrim(idsRaw, ",")
	if idsRaw == "" {
		return nil, errors.New("ids query parameter is required")
	}
	if len(ids) == 0 {
		return nil, errors.New("ids query parameter must include at least one valid id")
	}

	return ids, nil
}

func isSupportedCompareQueryKey(queryKey string) bool {
	switch queryKey {
	case "ids", "fields":
		return true
	default:
		return false
	}
}

func writeError(c echo.Context, status int, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"message": message,
			"status":  status,
		},
	})
}
