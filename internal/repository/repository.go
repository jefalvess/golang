//go:generate mockgen -source=repository.go -destination=repository_mock.go -package=repository

package repository

import (
	"comparify/internal/model"
	"context"
	"errors"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	ListByIDs(ctx context.Context, ids []string) ([]model.Product, error)
	GetByID(ctx context.Context, id string) (model.Product, error)
	GetSpecificationsByModel(ctx context.Context, modelName, productType string) (map[string]string, error)
	GetSpecificationsBatch(ctx context.Context, models []string, productType string) (map[string]map[string]string, error)
}
