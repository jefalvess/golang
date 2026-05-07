package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"comparify/internal/handler"
	"comparify/internal/service"
)

type stubService struct{}

func (s *stubService) GetItem(ctx context.Context, id, fields string) (map[string]any, error) {
	return map[string]any{"id": id}, nil
}

func (s *stubService) Compare(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

var _ service.Service = (*stubService)(nil)

func TestNewServer(t *testing.T) {
	h := handler.NewHandler(&stubService{})
	srv := NewServer(h)
	if srv == nil {
		t.Fatal("esperava servidor não-nulo")
	}
}

func TestNewEchoApplication(t *testing.T) {
	app := newEchoApplication()
	if app == nil {
		t.Fatal("esperava echo não-nulo")
	}
}

func TestRegisterRoutes_GetItem(t *testing.T) {
	h := handler.NewHandler(&stubService{})
	srv := NewServer(h)

	req := httptest.NewRequest(http.MethodGet, "/v1/items/p1", nil)
	rec := httptest.NewRecorder()
	srv.echo.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("rota GET /v1/items/:id não registrada, obteve 404")
	}
}

func TestRegisterRoutes_Compare(t *testing.T) {
	h := handler.NewHandler(&stubService{})
	srv := NewServer(h)

	req := httptest.NewRequest(http.MethodGet, "/v1/items/compare?ids=p1,p2", nil)
	rec := httptest.NewRecorder()
	srv.echo.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("rota GET /v1/items/compare não registrada, obteve 404")
	}
}
