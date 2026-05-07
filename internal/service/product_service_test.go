package service

import (
	"context"
	"errors"
	"testing"

	"comparify/internal/model"
	"comparify/internal/repository"
	"comparify/pkg/logger"
)

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

func TestGetItem_Success(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
		},
		specs: map[string]map[string]string{
			"celular:m1": {"batteryCapacity": "3000mAh"},
		},
	}
	svc := NewProductService(repo)
	item, err := svc.GetItem(context.Background(), "p1", "id,name,model")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if item["id"] != "p1" || item["name"] != "Produto 1" || item["model"] != "m1" {
		t.Errorf("campos básicos não batem: %v", item)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	repo := &mockRepository{products: map[string]model.Product{}}
	svc := NewProductService(repo)
	_, err := svc.GetItem(context.Background(), "naoexiste", "id")
	if err == nil {
		t.Fatal("esperava erro para produto inexistente")
	}
}

func TestCompare_Success(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
			"p2": {ID: "p2", Name: "Produto 2", Model: "m2", Type: "celular"},
		},
		specs: map[string]map[string]string{
			"celular:m1": {"batteryCapacity": "3000mAh"},
			"celular:m2": {"batteryCapacity": "4000mAh"},
		},
	}
	svc := NewProductService(repo)
	items, err := svc.Compare(context.Background(), []string{"p1", "p2"}, "id,model")
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("esperava 2 itens, obteve %d", len(items))
	}
	if items[0]["id"] != "p1" || items[1]["id"] != "p2" {
		t.Errorf("ordem dos ids não bate: %v", items)
	}
}

func TestCompare_InvalidField(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
		},
		specs: map[string]map[string]string{
			"celular:m1": {"batteryCapacity": "3000mAh"},
		},
	}
	svc := NewProductService(repo)
	_, err := svc.Compare(context.Background(), []string{"p1"}, "id,naocampo")
	if err == nil {
		t.Fatal("esperava erro para campo inválido")
	}
}

func TestCompare_NotFound(t *testing.T) {
	repo := &mockRepository{products: map[string]model.Product{}}
	svc := NewProductService(repo)
	items, err := svc.Compare(context.Background(), []string{"naoexiste"}, "id")
	if err == nil {
		t.Fatal("esperava erro para produto inexistente")
	}
	if items != nil {
		t.Errorf("esperava items nil, obteve %v", items)
	}
}

func TestGetItem_InvalidFieldSelection(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
		},
		specs: map[string]map[string]string{
			"celular:m1": {"batteryCapacity": "3000mAh"},
		},
	}
	svc := NewProductService(repo)
	_, err := svc.GetItem(context.Background(), "p1", "invalido")
	if err == nil {
		t.Fatal("esperava erro para campo inválido")
	}
	if !errors.Is(err, ErrInvalidFieldSelection) {
		t.Errorf("esperava ErrInvalidFieldSelection, obteve: %v", err)
	}
}

func TestGetItem_SpecsFetchFailure(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
		},
		specs: map[string]map[string]string{}, // chave ausente → GetSpecificationsByModel retorna erro
	}
	svc := NewProductService(repo)
	item, err := svc.GetItem(context.Background(), "p1", "id,name")
	if err != nil {
		t.Fatalf("esperava sucesso mesmo sem specs, obteve erro: %v", err)
	}
	if item["id"] != "p1" {
		t.Errorf("esperava id p1, obteve: %v", item["id"])
	}
}

func TestCompare_ErrProductNotFoundReturnsEmpty(t *testing.T) {
	repo := &mockRepository{
		products:     map[string]model.Product{},
		listByIDsErr: repository.ErrProductNotFound,
	}
	svc := NewProductService(repo)
	items, err := svc.Compare(context.Background(), []string{"naoexiste"}, "id")
	if err != nil {
		t.Fatalf("esperava lista vazia sem erro, obteve: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("esperava lista vazia, obteve %d itens", len(items))
	}
}

func TestCompare_SpecsBatchFailure(t *testing.T) {
	repo := &mockRepository{
		products: map[string]model.Product{
			"p1": {ID: "p1", Name: "Produto 1", Model: "m1", Type: "celular"},
		},
		specs:         map[string]map[string]string{},
		batchSpecsErr: errors.New("db error"),
	}
	svc := NewProductService(repo)
	items, err := svc.Compare(context.Background(), []string{"p1"}, "id,name")
	if err != nil {
		t.Fatalf("esperava sucesso mesmo com falha nas specs, obteve erro: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperava 1 item, obteve %d", len(items))
	}
}

func TestParseFields_EmptyAllowed(t *testing.T) {
	_, err := parseFields("id", map[string]any{})
	if err == nil {
		t.Fatal("esperava erro para campos permitidos vazios")
	}
}

func TestParseFields_EmptyFieldsAfterSplit(t *testing.T) {
	allowed := map[string]any{"id": "x"}
	_, err := parseFields(",", allowed)
	if err == nil {
		t.Fatal("esperava erro quando todos os campos são vazios após split")
	}
}

func TestEnsureModelVersion_EmptyModelName(t *testing.T) {
	specs := map[string]string{"battery": "3000mAh"}
	result := ensureModelVersion(specs, "")
	if result["battery"] != "3000mAh" {
		t.Errorf("esperava specs inalteradas, obteve: %v", result)
	}
	if _, exists := result["modelVersion"]; exists {
		t.Error("não esperava modelVersion quando modelName é vazio")
	}
}

func TestMain(m *testing.M) {
	logger.Init()
	m.Run()
}
