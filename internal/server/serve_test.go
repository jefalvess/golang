package server

import (
	"comparify/internal/handler"
	"comparify/internal/service"
	"comparify/pkg/logger"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

type stubService struct{}

func (s *stubService) ListItems(ctx context.Context) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

func (s *stubService) AvailableFields() []string {
	return []string{"id", "name", "price"}
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
			name:       "GET /v1/products registrado",
			method:     http.MethodGet,
			url:        "/v1/products",
			wantStatus: http.StatusOK,
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
