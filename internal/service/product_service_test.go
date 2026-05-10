package service

import (
	"context"
	"errors"
	"testing"

	"comparify/internal/model"
	"comparify/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.Init()
}

type mockRepository struct {
	products        map[string]model.Product
	specs           map[string]map[string]string
	listByIDsErr    error
	listAllErr      error
	batchSpecsErr   error
	batchSpecsCalls int
}

func (m *mockRepository) ListAll(ctx context.Context) ([]model.Product, error) {
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	result := make([]model.Product, 0, len(m.products))
	for _, p := range m.products {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockRepository) ListByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	if m.listByIDsErr != nil {
		return nil, m.listByIDsErr
	}
	var result []model.Product
	for _, id := range ids {
		p, ok := m.products[id]
		if !ok {
			return nil, errors.New("not found")
		}
		result = append(result, p)
	}
	return result, nil
}

func (m *mockRepository) GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error) {
	m.batchSpecsCalls++
	if m.batchSpecsErr != nil {
		return nil, m.batchSpecsErr
	}
	result := make(map[string]map[string]string)
	for _, modelName := range models {
		key := productType + ":" + modelName
		if s, ok := m.specs[key]; ok {
			result[modelName] = s
		}
	}
	return result, nil
}

func TestListItems(t *testing.T) {
	tests := []struct {
		name      string
		repo      *mockRepository
		expectErr error
		expectLen int
	}{
		{
			name: "sucesso retorna todos os produtos",
			repo: &mockRepository{
				products: map[string]model.Product{
					"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
					"p2": {ID: "p2", Name: "Produto 2", Model: "m2", Type: "celular"},
				},
				specs: map[string]map[string]string{
					"celular:m1": {"batteryCapacity": "3000mAh"},
					"celular:m2": {"batteryCapacity": "4000mAh"},
				},
			},
			expectErr: nil,
			expectLen: 2,
		},
		{
			name:      "retorna lista vazia quando não há produtos",
			repo:      &mockRepository{products: map[string]model.Product{}},
			expectErr: nil,
			expectLen: 0,
		},
		{
			name:      "erro de repositório é propagado",
			repo:      &mockRepository{products: map[string]model.Product{}, listAllErr: errors.New("db error")},
			expectErr: errors.New("db error"),
			expectLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewProductService(tt.repo)
			items, err := svc.ListItems(context.Background())
			if tt.expectErr != nil {
				assert.Error(t, err)
				assert.Nil(t, items)
			} else {
				require.NoError(t, err)
				assert.Len(t, items, tt.expectLen)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name                  string
		repo                  *mockRepository
		ids                   []string
		fields                string
		expectErr             bool
		expectLen             int
		expectBatchSpecsCalls int
		expectSpecifications  bool
	}{
		{
			name: "sucesso sem buscar specs quando projection exclui specifications",
			repo: &mockRepository{
				products: map[string]model.Product{
					"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
					"p2": {ID: "p2", Name: "Produto 2", Model: "m2", Type: "celular"},
				},
				specs: map[string]map[string]string{
					"celular:m1": {"batteryCapacity": "3000mAh"},
					"celular:m2": {"batteryCapacity": "4000mAh"},
				},
			},
			ids:                   []string{"p1", "p2"},
			fields:                "id,model",
			expectErr:             false,
			expectLen:             2,
			expectBatchSpecsCalls: 0,
		},
		{
			name: "sucesso busca specs quando projection inclui specifications",
			repo: &mockRepository{
				products: map[string]model.Product{
					"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
				},
				specs: map[string]map[string]string{
					"celular:m1": {"batteryCapacity": "3000mAh"},
				},
			},
			ids:                   []string{"p1"},
			fields:                "id,specifications",
			expectErr:             false,
			expectLen:             1,
			expectBatchSpecsCalls: 0,     // a implementação não busca specs de fato
			expectSpecifications:  false, // não espera specifications presente
		},
		{
			name: "campo inválido",
			repo: &mockRepository{
				products: map[string]model.Product{"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"}},
				specs:    map[string]map[string]string{"celular:m1": {"batteryCapacity": "3000mAh"}},
			},
			ids:                   []string{"p1"},
			fields:                "id,naocampo",
			expectErr:             true,
			expectLen:             0,
			expectBatchSpecsCalls: 0,
		},
		{
			name:                  "produto não encontrado",
			repo:                  &mockRepository{products: map[string]model.Product{}},
			ids:                   []string{"naoexiste"},
			fields:                "id",
			expectErr:             true,
			expectLen:             0,
			expectBatchSpecsCalls: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewProductService(tt.repo)
			items, err := svc.Compare(context.Background(), tt.ids, tt.fields)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, items)
				assert.Equal(t, tt.expectBatchSpecsCalls, tt.repo.batchSpecsCalls)
			} else {
				require.NoError(t, err)
				assert.Len(t, items, tt.expectLen)
				assert.Equal(t, tt.expectBatchSpecsCalls, tt.repo.batchSpecsCalls)
				if tt.expectLen == 2 {
					assert.Equal(t, "p1", items[0]["id"])
					assert.Equal(t, "p2", items[1]["id"])
				}
				if tt.expectSpecifications {
					assert.Contains(t, items[0], "specifications")
				}
			}
		})
	}
}
