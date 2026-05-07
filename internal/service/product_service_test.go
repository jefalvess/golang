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

func TestMain(m *testing.M) {
	logger.Init()
	m.Run()
}

type mockRepository struct {
	products      map[string]model.Product
	specs         map[string]map[string]string
	listByIDsErr  error
	batchSpecsErr error
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

func (m *mockRepository) GetByID(ctx context.Context, id string) (model.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return model.Product{}, errors.New("not found")
	}
	return p, nil
}

func (m *mockRepository) GetSpecificationsByModel(ctx context.Context, modelName, productType string) (map[string]string, error) {
	key := productType + ":" + modelName
	s, ok := m.specs[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *mockRepository) GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error) {
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

func TestGetItem(t *testing.T) {
	tests := []struct {
		name      string
		repo      *mockRepository
		id        string
		fields    string
		expectErr error
		expectMap map[string]interface{}
	}{
		{
			name: "sucesso",
			repo: &mockRepository{
				products: map[string]model.Product{"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"}},
				specs:    map[string]map[string]string{"celular:m1": {"batteryCapacity": "3000mAh"}},
			},
			id:        "p1",
			fields:    "id,name,model",
			expectErr: nil,
			expectMap: map[string]interface{}{"id": "p1", "name": "Produto 1", "model": "m1"},
		},
		{
			name:      "produto não encontrado",
			repo:      &mockRepository{products: map[string]model.Product{}},
			id:        "naoexiste",
			fields:    "id",
			expectErr: errors.New("not found"),
			expectMap: nil,
		},
		{
			name: "campo inválido",
			repo: &mockRepository{
				products: map[string]model.Product{"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"}},
				specs:    map[string]map[string]string{"celular:m1": {"batteryCapacity": "3000mAh"}},
			},
			id:        "p1",
			fields:    "invalido",
			expectErr: ErrInvalidFieldSelection,
			expectMap: nil,
		},
		{
			name: "sucesso_sem_specs",
			repo: &mockRepository{
				products: map[string]model.Product{"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"}},
				specs:    map[string]map[string]string{},
			},
			id:        "p1",
			fields:    "id,name",
			expectErr: nil,
			expectMap: map[string]interface{}{"id": "p1", "name": "Produto 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewProductService(tt.repo)
			item, err := svc.GetItem(context.Background(), tt.id, tt.fields)
			if tt.expectErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectErr, ErrInvalidFieldSelection) {
					assert.ErrorIs(t, err, ErrInvalidFieldSelection)
				}
				assert.Nil(t, item)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectMap, item)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name      string
		repo      *mockRepository
		ids       []string
		fields    string
		expectErr bool
		expectLen int
	}{
		{
			name: "sucesso",
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
			ids:       []string{"p1", "p2"},
			fields:    "id,model",
			expectErr: false,
			expectLen: 2,
		},
		{
			name: "campo inválido",
			repo: &mockRepository{
				products: map[string]model.Product{"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"}},
				specs:    map[string]map[string]string{"celular:m1": {"batteryCapacity": "3000mAh"}},
			},
			ids:       []string{"p1"},
			fields:    "id,naocampo",
			expectErr: true,
			expectLen: 0,
		},
		{
			name:      "produto não encontrado",
			repo:      &mockRepository{products: map[string]model.Product{}},
			ids:       []string{"naoexiste"},
			fields:    "id",
			expectErr: true,
			expectLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewProductService(tt.repo)
			items, err := svc.Compare(context.Background(), tt.ids, tt.fields)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, items)
			} else {
				require.NoError(t, err)
				assert.Len(t, items, tt.expectLen)
				if tt.expectLen == 2 {
					assert.Equal(t, "p1", items[0]["id"])
					assert.Equal(t, "p2", items[1]["id"])
				}
			}
		})
	}
}
