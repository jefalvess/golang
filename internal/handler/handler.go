package handler

import (
	"comparify/internal/service"
	"errors"
	"net/http"

	"comparify/internal/repository"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	Service *service.ProductService
}

func NewHandler(svc *service.ProductService) *Handler {
	return &Handler{Service: svc}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetItem(c echo.Context) error {
	itemID := c.Param("id")
	if itemID == "" {
		return writeError(c, http.StatusNotFound, "item not found")
	}

	item, fields, err := h.Service.GetItem(itemID, c.QueryParam("fields"))
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
		"item":            item,
		"requestedFields": fields,
	})
}

func (h *Handler) Compare(c echo.Context) error {
	filters := make(map[string]string)
	for _, key := range []string{"brand", "color"} {
		if v := c.QueryParam(key); v != "" {
			filters[key] = v
		}
	}

	items, fields, err := h.Service.Compare(filters, c.QueryParam("fields"))
	if err != nil {
		if errors.Is(err, service.ErrInvalidFieldSelection) {
			return writeError(c, http.StatusBadRequest, err.Error())
		}

		return writeError(c, http.StatusInternalServerError, "failed to load items")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"items":           items,
		"requestedFields": fields,
		"count":           len(items),
	})
}

func writeError(c echo.Context, status int, message string) error {
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"message": message,
			"status":  status,
		},
	})
}
