package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"comparify/internal/handler"
	"comparify/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Run("servidor não-nulo", func(t *testing.T) {
		h := handler.NewHandler(&stubService{})
		srv := NewServer(h)
		require.NotNil(t, srv)
	})
}

func TestNewEchoApplication(t *testing.T) {
	t.Run("echo não-nulo", func(t *testing.T) {
		app := newEchoApplication()
		require.NotNil(t, app)
	})
}

func TestRegisterRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{
			name:       "GET /v1/products/:id registrado",
			method:     http.MethodGet,
			url:        "/v1/products/p1",
			wantStatus: http.StatusOK, // espera-se sucesso do stub
		},
		{
			name:       "GET /v1/products/compare registrado",
			method:     http.MethodGet,
			url:        "/v1/products/compare?ids=p1,p2",
			wantStatus: http.StatusOK, // espera-se sucesso do stub
		},
	}
	h := handler.NewHandler(&stubService{})
	srv := NewServer(h)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rec := httptest.NewRecorder()
			srv.echo.ServeHTTP(rec, req)
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "rota não registrada, obteve 404")
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
