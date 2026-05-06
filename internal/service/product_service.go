package service

import (
	"comparify/internal/repository"
	"comparify/pkg/logger"
	"comparify/pkg/utils"
	"errors"
	"sort"
	"sync"
	"time"
)

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
	specs, err := s.Repo.GetSpecificationsByType(product.ID, product.SpecsTable)
	if err != nil {
		logger.Logger.Warnw("GetItem: failed to fetch specifications",
			"productID", product.ID,
			"specsTable", product.SpecsTable,
			"error", err,
		)
	} else {
		product.Specifications = specs
	}
	fields, err := parseFields(fieldsRaw, product.FieldMap())
	if err != nil {
		logger.Logger.Warnw("GetItem parseFields error",
			"error", err,
		)
		return nil, nil, err
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
		return nil, nil, err
	}
	items := make([]map[string]any, 0, len(products))

	// Agrupa IDs por tipo para buscar specs em batch (evita N+1)
	productIDsByType := make(map[string][]string)
	for _, product := range products {
		productIDsByType[product.SpecsTable] = append(productIDsByType[product.SpecsTable], product.ID)
	}

	// Busca specs de cada tipo em paralelo via goroutines + channel
	type specsResult struct {
		specs       map[string]map[string]string
		productType string
		err         error
	}
	ch := make(chan specsResult, len(productIDsByType))
	var wg sync.WaitGroup
	for productType, productIDs := range productIDsByType {
		wg.Add(1)
		go func(pt string, ids []string) {
			defer wg.Done()
			specsBatch, err := s.Repo.GetSpecificationsBatch(ids, pt)
			ch <- specsResult{specs: specsBatch, productType: pt, err: err}
		}(productType, productIDs)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	specsByProductID := make(map[string]map[string]string)
	for res := range ch {
		if res.err != nil {
			logger.Logger.Warnw("Compare: failed to fetch specs batch",
				"productType", res.productType,
				"error", res.err,
			)
			continue
		}
		for fetchedProductID, fetchedSpecs := range res.specs {
			specsByProductID[fetchedProductID] = fetchedSpecs
		}
	}

	for _, product := range products {
		productMap := product.FieldMap()
		productMap["specifications"] = specsByProductID[product.ID]
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

// parseFields e selectFields são helpers copiados do handler
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
	parts := splitAndTrim(raw, ",")
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

func splitAndTrim(s, sep string) []string {
	return utils.SplitAndTrim(s, sep)
}
