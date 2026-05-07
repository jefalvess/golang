package service

import (
	"comparify/internal/repository"
	"comparify/pkg/logger"
	"comparify/pkg/utils"
	"errors"
	"fmt"
	"sort"
	"time"
)

var ErrInvalidFieldSelection = errors.New("invalid field selection")

type ProductService struct {
	Repo repository.Repository
}

func NewProductService(repo repository.Repository) *ProductService {
	return &ProductService{Repo: repo}
}

// GetItem retorna um produto por ID e os campos projetados
func (s *ProductService) GetItem(id string, fieldsRaw string) (map[string]any, []string, error) {
	start := time.Now()
	logger.Logger.Infow("GetItem called",
		"id", id,
		"fieldsRaw", fieldsRaw,
	)
	product, err := s.Repo.GetByID(id)
	if err != nil {
		logger.Logger.Errorw("GetItem error",
			"error", err,
		)
		return nil, nil, err
	}
	specs, err := s.Repo.GetSpecificationsByModel(product.Model, product.Type)
	if err != nil {
		logger.Logger.Warnw("GetItem: failed to fetch specifications",
			"productID", product.ID,
			"model", product.Model,
			"productType", product.Type,
			"error", err,
		)
	} else {
		if _, exists := specs["modelVersion"]; !exists && product.Model != "" {
			specs["modelVersion"] = product.Model
		}
		product.Specifications = specs
	}
	fields, err := parseFields(fieldsRaw, product.FieldMap())
	if err != nil {
		logger.Logger.Warnw("GetItem parseFields error",
			"error", err,
		)
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidFieldSelection, err)
	}
	elapsed := time.Since(start)
	logger.Logger.Infow("GetItem success",
		"id", id,
		"fields", fields,
		"duration", elapsed,
	)
	return selectFields(product.FieldMap(), fields), fields, nil
}

// Compare retorna produtos filtrados e os campos projetados
func (s *ProductService) Compare(filters map[string]string, fieldsRaw string) ([]map[string]any, []string, error) {
	start := time.Now()
	logger.Logger.Infow("Compare called",
		"filters", filters,
		"fieldsRaw", fieldsRaw,
	)
	products, err := s.Repo.ListByFilters(filters)
	if err != nil && !errors.Is(err, repository.ErrProductNotFound) {
		logger.Logger.Errorw("Compare error",
			"error", err,
		)
		return nil, nil, err
	}
	if len(products) == 0 {
		logger.Logger.Warnw("Compare: no products found",
			"filters", filters,
		)
		return []map[string]any{}, []string{}, nil
	}
	fields, err := parseFields(fieldsRaw, products[0].FieldMap())
	if err != nil {
		logger.Logger.Warnw("Compare parseFields error",
			"error", err,
		)
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidFieldSelection, err)
	}
	items := make([]map[string]any, 0, len(products))

	// Agrupa modelos por tabela de specs para reutilizar a mesma linha entre produtos iguais.
	modelsByType := make(map[string][]string)
	seenModelsByType := make(map[string]map[string]struct{})
	for _, product := range products {
		if _, ok := seenModelsByType[product.Type]; !ok {
			seenModelsByType[product.Type] = make(map[string]struct{})
		}
		if _, exists := seenModelsByType[product.Type][product.Model]; exists {
			continue
		}
		seenModelsByType[product.Type][product.Model] = struct{}{}
		modelsByType[product.Type] = append(modelsByType[product.Type], product.Model)
	}

	specsByModelAndType := make(map[string]map[string]string)
	for productType, models := range modelsByType {
		specsBatch, batchErr := s.Repo.GetSpecificationsBatch(models, productType)
		if batchErr != nil {
			logger.Logger.Warnw("Compare: failed to fetch specs batch",
				"productType", productType,
				"error", batchErr,
			)
			continue
		}
		for modelName, fetchedSpecs := range specsBatch {
			if _, exists := fetchedSpecs["modelVersion"]; !exists && modelName != "" {
				fetchedSpecs["modelVersion"] = modelName
			}
			specsByModelAndType[productType+"\x00"+modelName] = fetchedSpecs
		}
	}

	for _, product := range products {
		productMap := product.FieldMap()
		productMap["specifications"] = specsByModelAndType[product.Type+"\x00"+product.Model]
		items = append(items, selectFields(productMap, fields))
	}
	elapsed := time.Since(start)
	logger.Logger.Infow("Compare success",
		"filters", filters,
		"fields", fields,
		"count", len(items),
		"duration", elapsed,
	)
	return items, fields, nil
}

func parseFields(raw string, allowed map[string]any) ([]string, error) {
	if len(allowed) == 0 {
		return nil, errors.New("no allowed fields")
	}
	if raw == "" {
		fields := make([]string, 0, len(allowed))
		for field := range allowed {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		return fields, nil
	}
	parts := utils.SplitAndTrim(raw, ",")
	fields := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, field := range parts {
		if _, ok := allowed[field]; !ok {
			return nil, errors.New("unsupported field: " + field)
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return nil, errors.New("fields query parameter must include at least one valid field")
	}

	return fields, nil
}

func selectFields(product map[string]any, fields []string) map[string]any {
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		selected[field] = product[field]
	}
	return selected
}
