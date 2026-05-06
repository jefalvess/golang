package handler

import (
	"comparify/internal/service"
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct {
	Service *service.ProductService
}

func NewHandler(svc *service.ProductService) *Handler {
	return &Handler{Service: svc}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	item, fields, err := h.Service.GetItem(id, r.URL.Query().Get("fields"))
	if err != nil {
		// Aqui assume-se que o service retorna erro de not found já tratado
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"item":            item,
		"requestedFields": fields,
	})
}

func (h *Handler) Compare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := r.URL.Query()
	filters := make(map[string]string)
	for _, key := range []string{"brand", "color"} {
		if v := query.Get(key); v != "" {
			filters[key] = v
		}
	}
	items, fields, err := h.Service.Compare(filters, query.Get("fields"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load items")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":           items,
		"requestedFields": fields,
		"count":           len(items),
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"status":  status,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
