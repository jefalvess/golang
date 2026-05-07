package repository

import (
	"comparify/internal/model"
	"errors"
	"fmt"
	"strings"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	ListByFilters(filters map[string]string) ([]model.Product, error)
	GetByID(id string) (model.Product, error)
	GetSpecificationsByModel(modelName, productType string) (map[string]string, error)
	GetSpecificationsBatch(models []string, productType string) (map[string]map[string]string, error)
}

type InMemoryRepository struct {
	products map[string]model.Product
	order    []string
}

func NewInMemoryRepository(products []model.Product) *InMemoryRepository {
	repo := &InMemoryRepository{
		products: make(map[string]model.Product, len(products)),
		order:    make([]string, 0, len(products)),
	}

	for _, product := range products {
		if product.Model == "" {
			product.Model = firstNonEmpty(product.Specifications["modelVersion"], product.Name)
		}
		repo.products[product.ID] = product
		repo.order = append(repo.order, product.ID)
	}

	return repo
}

func (r *InMemoryRepository) ListByFilters(filters map[string]string) ([]model.Product, error) {
	var result []model.Product
	for _, productID := range r.order {
		candidate := r.products[productID]
		allFiltersMatch := true
		for filterKey, filterValue := range filters {
			acceptedValues := strings.Split(filterValue, ",")
			productFieldValue := ""
			switch filterKey {
			case "color":
				productFieldValue = candidate.Color
			case "brand":
				productFieldValue = candidate.Specifications["brand"]
			}
			valueFound := false
			for _, acceptedValue := range acceptedValues {
				if strings.EqualFold(strings.TrimSpace(acceptedValue), productFieldValue) {
					valueFound = true
					break
				}
			}
			if !valueFound {
				allFiltersMatch = false
				break
			}
		}
		if allFiltersMatch {
			result = append(result, candidate)
		}
	}
	if len(result) == 0 {
		return nil, ErrProductNotFound
	}
	return result, nil
}

func (r *InMemoryRepository) ListByIDs(ids []string) ([]model.Product, error) {
	products := make([]model.Product, 0, len(ids))
	for _, id := range ids {
		product, err := r.GetByID(id)
		if err != nil {
			return nil, fmt.Errorf("lookup %q: %w", id, err)
		}
		products = append(products, product)
	}

	return products, nil
}

func (r *InMemoryRepository) GetByID(id string) (model.Product, error) {
	product, ok := r.products[id]
	if !ok {
		return model.Product{}, ErrProductNotFound
	}

	return product, nil
}

func (r *InMemoryRepository) GetSpecificationsByModel(modelName, productType string) (map[string]string, error) {
	for _, productID := range r.order {
		product := r.products[productID]
		if strings.EqualFold(product.Model, modelName) && strings.EqualFold(product.Type, productType) {
			return product.Specifications, nil
		}
	}

	return nil, ErrProductNotFound
}

func (r *InMemoryRepository) GetSpecificationsBatch(models []string, productType string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string, len(models))
	for _, modelName := range models {
		specifications, err := r.GetSpecificationsByModel(modelName, productType)
		if err == nil {
			result[modelName] = specifications
		}
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
