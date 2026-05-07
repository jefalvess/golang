package handler

import (
	"comparify/internal/repository"
	"comparify/internal/service"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Testa os cenários de busca de produto por ID usando table-driven test e testify
func TestGetItem(t *testing.T) {
	type fields struct {
		getItemFunc func(ctx context.Context, id, fields string) (map[string]any, error)
	}
	tests := []struct {
		name       string
		fields     fields
		paramID    string
		query      string
		wantStatus int
		wantInBody string
		wantErr    bool
	}{
		{
			name: "sucesso ao buscar produto existente",
			fields: fields{getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
				return map[string]any{"id": id, "name": "Produto Teste"}, nil
			}},
			paramID:    "p1",
			query:      "?fields=id,name",
			wantStatus: http.StatusOK,
			wantInBody: "Produto Teste",
			wantErr:    false,
		},
		{
			name: "erro 404 ao buscar produto inexistente",
			fields: fields{getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
				return nil, repository.ErrProductNotFound
			}},
			paramID:    "naoexiste",
			query:      "",
			wantStatus: http.StatusNotFound,
			wantInBody: "item not found",
			wantErr:    false,
		},
		{
			name: "erro 400 ao buscar produto com campo inválido",
			fields: fields{getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
				return nil, service.ErrInvalidFieldSelection
			}},
			paramID:    "p1",
			query:      "?fields=invalido",
			wantStatus: http.StatusBadRequest,
			wantInBody: "",
			wantErr:    false,
		},
		{
			name: "erro 500 ao ocorrer erro inesperado no serviço",
			fields: fields{getItemFunc: func(ctx context.Context, id, fields string) (map[string]any, error) {
				return nil, errors.New("unexpected db error")
			}},
			paramID:    "p1",
			query:      "",
			wantStatus: http.StatusInternalServerError,
			wantInBody: "",
			wantErr:    false,
		},
		{
			name:       "erro 404 ao buscar produto sem ID",
			fields:     fields{getItemFunc: nil},
			paramID:    "",
			query:      "",
			wantStatus: http.StatusNotFound,
			wantInBody: "",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var svc service.Service
			if tt.fields.getItemFunc != nil {
				svc = &mockService{getItemFunc: tt.fields.getItemFunc}
			} else {
				svc = &mockService{}
			}
			h := NewHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/v1/products/"+tt.paramID+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.paramID)

			err := h.GetItem(c)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				if tt.wantInBody != "" {
					assert.Contains(t, rec.Body.String(), tt.wantInBody)
				}
			}
		})
	}
}

// Testa os cenários de comparação de produtos usando table-driven test e testify
func TestCompare(t *testing.T) {
	type fields struct {
		compareFunc func(ctx context.Context, ids []string, fields string) ([]map[string]any, error)
	}
	tests := []struct {
		name       string
		fields     fields
		query      string
		wantStatus int
		wantInBody []string
		wantErr    bool
	}{
		{
			name: "sucesso ao comparar dois produtos válidos",
			fields: fields{compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
				return []map[string]any{{"id": "p1"}, {"id": "p2"}}, nil
			}},
			query:      "?ids=p1,p2&fields=id",
			wantStatus: http.StatusOK,
			wantInBody: []string{"p1", "p2"},
			wantErr:    false,
		},
		{
			name: "erro 400 ao passar parâmetro inválido na comparação",
			fields: fields{compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
				return nil, service.ErrInvalidFieldSelection
			}},
			query:      "?foo=bar",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"unsupported compare query parameter"},
			wantErr:    false,
		},
		{
			name:       "erro 400 ao omitir ids na comparação",
			fields:     fields{compareFunc: nil},
			query:      "",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"ids query parameter is required"},
			wantErr:    false,
		},
		{
			name:       "erro 400 ao passar ids vazios na comparação",
			fields:     fields{compareFunc: nil},
			query:      "?ids=,",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"must include at least one"},
			wantErr:    false,
		},
		{
			name: "erro 404 ao comparar produtos não encontrados",
			fields: fields{compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
				return nil, repository.ErrProductNotFound
			}},
			query:      "?ids=p1,p2",
			wantStatus: http.StatusNotFound,
			wantInBody: nil,
			wantErr:    false,
		},
		{
			name: "erro 500 ao ocorrer erro inesperado na comparação",
			fields: fields{compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
				return nil, errors.New("unexpected db error")
			}},
			query:      "?ids=p1,p2",
			wantStatus: http.StatusInternalServerError,
			wantInBody: nil,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var svc service.Service
			if tt.fields.compareFunc != nil {
				svc = &mockService{compareFunc: tt.fields.compareFunc}
			} else {
				svc = &mockService{}
			}
			h := NewHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/v1/products/compare"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.Compare(c)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				for _, want := range tt.wantInBody {
					assert.Contains(t, rec.Body.String(), want)
				}
			}
		})
	}
}
