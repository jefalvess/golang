package repository

import (
	"comparify/internal/model"
	"context"
	"testing"
)

type fakeRepo struct{}

func (f *fakeRepo) ListByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	return nil, nil
}
func (f *fakeRepo) GetByID(ctx context.Context, id string) (model.Product, error) {
	return model.Product{}, nil
}
func (f *fakeRepo) GetSpecificationsByModel(ctx context.Context, modelName, productType string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeRepo) GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error) {
	return nil, nil
}

func TestRepository_Interface(t *testing.T) {
	var _ Repository = (*fakeRepo)(nil)
}
