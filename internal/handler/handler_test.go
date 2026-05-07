package handler

import (
	"comparify/internal/repository"
	"comparify/internal/service"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type mockService struct {
	getItemFunc func(ctx context.Context, id, fields string) (map[string]any, error)
	compareFunc func(ctx context.Context, ids []string, fields string) ([]map[string]any, error)
}

func (m *mockService) GetItem(ctx context.Context, id, fields string) (map[string]any, error) {
	return m.getItemFunc(ctx, id, fields)
}
func (m *mockService) Compare(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
	return m.compareFunc(ctx, ids, fields)
}

var _ service.Service = (*mockService)(nil)

func TestGetItem_Success(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
			return map[string]any{"id": id, "name": "Produto Teste"}, nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/p1?fields=id,name", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("p1")

	err := h.GetItem(c)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "Produto Teste") {
		t.Errorf("esperava resposta com Produto Teste, obteve: %s", rec.Body.String())
	}
}

func TestGetItem_NotFound(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/naoexiste", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("naoexiste")

	err := h.GetItem(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperava status 404, obteve %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "item not found") {
		t.Errorf("esperava mensagem 'item not found', obteve: %s", rec.Body.String())
	}
}

func TestCompare_Success(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
			return []map[string]any{{"id": ids[0]}, {"id": ids[1]}}, nil
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare?ids=p1,p2&fields=id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "p1") || !strings.Contains(rec.Body.String(), "p2") {
		t.Errorf("esperava ids p1 e p2 na resposta, obteve: %s", rec.Body.String())
	}
}

func TestCompare_InvalidParam(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
			return nil, service.ErrInvalidFieldSelection
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare?foo=bar", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unsupported compare query parameter") {
		t.Errorf("esperava mensagem 'unsupported compare query parameter', obteve: %s", rec.Body.String())
	}
}

func TestGetItem_EmptyID(t *testing.T) {
	e := echo.New()
	svc := &mockService{}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("")

	err := h.GetItem(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperava status 404, obteve %d", rec.Code)
	}
}

func TestGetItem_InvalidField(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
			return nil, service.ErrInvalidFieldSelection
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/p1?fields=invalido", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("p1")

	err := h.GetItem(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", rec.Code)
	}
}

func TestGetItem_InternalError(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/p1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("p1")

	err := h.GetItem(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("esperava status 500, obteve %d", rec.Code)
	}
}

func TestCompare_MissingIds(t *testing.T) {
	e := echo.New()
	svc := &mockService{}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ids query parameter is required") {
		t.Errorf("esperava mensagem sobre ids obrigatório, obteve: %s", rec.Body.String())
	}
}

func TestCompare_EmptyIdsAfterTrim(t *testing.T) {
	e := echo.New()
	svc := &mockService{}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare?ids=,", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "must include at least one") {
		t.Errorf("esperava mensagem sobre ids inválidos, obteve: %s", rec.Body.String())
	}
}

func TestCompare_ServiceNotFound(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
			return nil, repository.ErrProductNotFound
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare?ids=p1,p2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperava status 404, obteve %d", rec.Code)
	}
}

func TestCompare_InternalError(t *testing.T) {
	e := echo.New()
	svc := &mockService{
		compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/products/compare?ids=p1,p2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Compare(c)
	if err != nil {
		t.Fatalf("esperava resposta HTTP, obteve erro: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("esperava status 500, obteve %d", rec.Code)
	}
}
