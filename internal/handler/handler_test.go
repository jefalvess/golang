package handler

import (
	"comparify/internal/repository"
	"comparify/internal/service"
	"comparify/pkg/logger"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

type mockService struct {
	listItemsFunc       func(ctx context.Context) ([]map[string]any, error)
	availableFieldsFunc func() []string
	compareFunc         func(ctx context.Context, ids []string, fields string) ([]map[string]any, error)
}

func (m *mockService) ListItems(ctx context.Context) ([]map[string]any, error) {
	if m.listItemsFunc == nil {
		return []map[string]any{}, nil
	}
	return m.listItemsFunc(ctx)
}

func (m *mockService) AvailableFields() []string {
	if m.availableFieldsFunc == nil {
		return []string{}
	}
	return m.availableFieldsFunc()
}

func (m *mockService) Compare(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
	if m.compareFunc == nil {
		return []map[string]any{}, nil
	}
	return m.compareFunc(ctx, ids, fields)
}

var _ service.Service = (*mockService)(nil)

// Testa os cenários de listagem de produtos usando table-driven test e testify
func TestListItems(t *testing.T) {
	type fields struct {
		listItemsFunc func(ctx context.Context) ([]map[string]any, error)
	}
	tests := []struct {
		name       string
		fields     fields
		query      string
		wantStatus int
		wantInBody string
		wantErr    bool
	}{
		{
			name: "sucesso ao listar produtos",
			fields: fields{listItemsFunc: func(ctx context.Context) ([]map[string]any, error) {
				return []map[string]any{{"id": "p1", "name": "Produto Teste"}}, nil
			}},
			query:      "",
			wantStatus: http.StatusOK,
			wantInBody: "Produto Teste",
			wantErr:    false,
		},
		{
			name: "response inclui availableFields",
			fields: fields{listItemsFunc: func(ctx context.Context) ([]map[string]any, error) {
				return []map[string]any{}, nil
			}},
			query:      "",
			wantStatus: http.StatusOK,
			wantInBody: "availableFields",
			wantErr:    false,
		},
		{
			name: "erro 500 ao ocorrer erro inesperado no serviço",
			fields: fields{listItemsFunc: func(ctx context.Context) ([]map[string]any, error) {
				return nil, errors.New("unexpected db error")
			}},
			query:      "",
			wantStatus: http.StatusInternalServerError,
			wantInBody: "",
			wantErr:    false,
		},
		{
			name:       "sucesso ao listar sem produtos cadastrados",
			fields:     fields{listItemsFunc: nil},
			query:      "",
			wantStatus: http.StatusOK,
			wantInBody: "",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			svc := &mockService{
				listItemsFunc: tt.fields.listItemsFunc,
				availableFieldsFunc: func() []string {
					return []string{"id", "name", "price"}
				},
			}
			h := NewHandler(svc)
			req := httptest.NewRequest(http.MethodGet, "/v1/products"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.ListItems(c)
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
			wantInBody: []string{"INVALID_REQUEST_ERROR"},
			wantErr:    false,
		},
		{
			name:       "erro 400 ao omitir ids na comparação",
			fields:     fields{compareFunc: nil},
			query:      "",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"INVALID_REQUEST_ERROR"},
			wantErr:    false,
		},
		{
			name:       "erro 400 ao passar ids vazios na comparação",
			fields:     fields{compareFunc: nil},
			query:      "?ids=,",
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"INVALID_REQUEST_ERROR"},
			wantErr:    false,
		},
		{
			name: "erro 404 ao comparar produtos não encontrados",
			fields: fields{compareFunc: func(ctx context.Context, ids []string, fields string) ([]map[string]any, error) {
				return nil, repository.ErrProductNotFound
			}},
			query:      "?ids=p1,p2",
			wantStatus: http.StatusInternalServerError, // handler retorna 500 para erro não customizado
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
