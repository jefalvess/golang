package service

import (
	"comparify/internal/repository"
	"comparify/pkg/logger"
	"comparify/pkg/utils"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

var ErrInvalidFieldSelection = errors.New("invalid field selection")

type ProductService struct {
	Repo repository.Repository
}

type specificationKey struct {
	productType string
	modelName   string
}

func NewProductService(repo repository.Repository) *ProductService {
	return &ProductService{Repo: repo}
}

// GetItem retorna um produto por ID com os campos projetados.
func (s *ProductService) GetItem(ctx context.Context, id string, fieldsRaw string) (map[string]any, error) {
	start := time.Now()
	logger.Logger.Infow("GetItem called",
		"id", id,
		"fieldsRaw", fieldsRaw,
	)
	product, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		logger.Logger.Errorw("GetItem error",
			"error", err,
		)
		return nil, err
	}
	specs, err := s.Repo.GetSpecificationsByModel(ctx, product.Model, product.Type)
	if err != nil {
		logger.Logger.Warnw("GetItem: failed to fetch specifications",
			"productID", product.ID,
			"model", product.Model,
			"productType", product.Type,
			"error", err,
		)
	} else {
		product.Specifications = ensureModelVersion(specs, product.Model)
	}
	fields, err := parseFields(fieldsRaw, product.FieldMap())
	if err != nil {
		logger.Logger.Warnw("GetItem parseFields error",
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrInvalidFieldSelection, err)
	}
	elapsed := time.Since(start)
	logger.Logger.Infow("GetItem success",
		"id", id,
		"fields", fields,
		"duration", elapsed,
	)
	return selectFields(product.FieldMap(), fields), nil
}

// Compare retorna itens específicos selecionados por ids com os campos projetados.
func (s *ProductService) Compare(ctx context.Context, ids []string, fieldsRaw string) ([]map[string]any, error) {
	start := time.Now()
	logger.Logger.Infow("Compare called",
		"ids", ids,
		"fieldsRaw", fieldsRaw,
	)
	products, err := s.Repo.ListByIDs(ctx, ids)
	if err != nil && !errors.Is(err, repository.ErrProductNotFound) {
		logger.Logger.Errorw("Compare error",
			"error", err,
		)
		return nil, err
	}
	if len(products) == 0 {
		logger.Logger.Warnw("Compare: no products found", "ids", ids)
		return []map[string]any{}, nil
	}
	fields, err := parseFields(fieldsRaw, products[0].FieldMap())
	if err != nil {
		logger.Logger.Warnw("Compare parseFields error",
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrInvalidFieldSelection, err)
	}
	items := make([]map[string]any, 0, len(products))

	// Agrupa por tipo e modelo para buscar specs uma vez só, mesmo quando mais de um
	// item comparado reutiliza a mesma linha de especificação.
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

	specsByModelAndType := make(map[specificationKey]map[string]string)
	for productType, models := range modelsByType {
		specsBatch, batchErr := s.Repo.GetSpecificationsBatch(ctx, models, productType)
		if batchErr != nil {
			logger.Logger.Warnw("Compare: failed to fetch specs batch",
				"productType", productType,
				"error", batchErr,
			)
			continue
		}
		for modelName, fetchedSpecs := range specsBatch {
			specsByModelAndType[specificationKey{productType: productType, modelName: modelName}] = ensureModelVersion(fetchedSpecs, modelName)
		}
	}

	for _, product := range products {
		productMap := product.FieldMap()
		productMap["specifications"] = specsByModelAndType[specificationKey{productType: product.Type, modelName: product.Model}]
		items = append(items, selectFields(productMap, fields))
	}
	elapsed := time.Since(start)
	logger.Logger.Infow("Compare success",
		"ids", ids,
		"fields", fields,
		"count", len(items),
		"duration", elapsed,
	)
	return items, nil
}

// parseFields valida a projeção pedida pelo cliente contra os campos expostos pela API.
// Quando nenhum campo é informado, devolve todos os campos disponíveis em ordem estável.
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

// selectFields aplica a projeção já validada sem reprocessar as regras de autorização.
func selectFields(product map[string]any, fields []string) map[string]any {
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		selected[field] = product[field]
	}
	return selected
}

// ensureModelVersion completa a resposta com modelVersion quando o seed usa model na
// tabela principal, mas a linha de especificações não repete esse valor.
func ensureModelVersion(specifications map[string]string, modelName string) map[string]string {
	if specifications == nil {
		return nil
	}
	if _, exists := specifications["modelVersion"]; exists || modelName == "" {
		return specifications
	}

	completed := make(map[string]string, len(specifications)+1)
	for key, value := range specifications {
		completed[key] = value
	}
	completed["modelVersion"] = modelName

	return completed
}
