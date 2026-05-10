package service

import (
	"comparify/internal/model"
	"comparify/internal/repository"
	"comparify/pkg/customerror"
	"comparify/pkg/utils"
	"context"
	"errors"
	"fmt"
	"sort"
)

// Erros sentinela de domínio — mapeados para HTTP exclusivamente no handler.
var (
	ErrProductNotFound       = errors.New("product not found")
	ErrInvalidFieldSelection = errors.New("invalid field selection")
	ErrInvalidQueryParam     = errors.New("invalid query parameter")
)

const specificationsField = "specifications"

type ProductService struct {
	Repo repository.Repository
}

func NewProductService(repo repository.Repository) *ProductService {
	return &ProductService{Repo: repo}
}

// AvailableFields retorna os campos de primeiro nível projetáveis pela API.
func (s *ProductService) AvailableFields() []string {
	fieldMap := model.Product{}.FieldMap()
	fields := make([]string, 0, len(fieldMap))
	for fieldName := range fieldMap {
		fields = append(fields, fieldName)
	}
	sort.Strings(fields)
	return fields
}

// ListItems retorna todos os produtos com todos os campos disponíveis.
func (s *ProductService) ListItems(ctx context.Context) ([]map[string]any, error) {
	const componentName = "ProductService.ListItems"

	products, err := s.Repo.ListAll(ctx)
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}

	items := make([]map[string]any, len(products))
	for idx, product := range products {
		fieldMap := product.FieldMap()
		fieldMap[specificationsField] = product.Specifications
		items[idx] = fieldMap
	}
	return items, nil
}

// Compare retorna itens específicos selecionados por ids com os campos projetados.
func (s *ProductService) Compare(ctx context.Context, ids []string, fieldsQuery string) ([]map[string]any, error) {
	const componentName = "ProductService.Compare"

	products, err := s.Repo.ListByIDs(ctx, ids)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, customerror.ThrowNew(componentName, customerror.NotFoundError, fmt.Errorf("%w: ids %v", ErrProductNotFound, ids))
		}
		return nil, customerror.ThrowNew(componentName, customerror.RequestExecutionError, err)
	}
	if len(products) == 0 {
		return nil, customerror.ThrowNew(componentName, customerror.NotFoundError, fmt.Errorf("%w: ids %v", ErrProductNotFound, ids))
	}

	fields, err := parseRequestedFields(fieldsQuery, products[0].FieldMap())
	if err != nil {
		return nil, customerror.ThrowNew(componentName, customerror.InvalidRequestError, fmt.Errorf("%w: %v", ErrInvalidFieldSelection, err))
	}

	items := make([]map[string]any, len(products))
	for idx, product := range products {
		productFields := product.FieldMap()
		// Só adicionamos specs ao mapa público quando elas fazem parte da projeção final.
		if containsField(fields, specificationsField) {
			productFields[specificationsField] = product.Specifications
		}
		items[idx] = projectFields(productFields, fields)
	}
	return items, nil
}

// parseRequestedFields valida e retorna os campos projetados, preservando a ordem solicitada.
func parseRequestedFields(requested string, allowed map[string]any) ([]string, error) {
	if len(allowed) == 0 {
		return nil, errors.New("no allowed fields")
	}
	if requested == "" {
		// Sem fields explícito, o contrato retorna todos os campos em ordem estável.
		fields := make([]string, 0, len(allowed))
		for fieldName := range allowed {
			fields = append(fields, fieldName)
		}
		sort.Strings(fields)
		return fields, nil
	}
	fieldTokens := utils.SplitAndTrim(requested, ",")
	fields := make([]string, 0, len(fieldTokens))
	seen := make(map[string]struct{}, len(fieldTokens))
	for _, fieldName := range fieldTokens {
		if _, ok := allowed[fieldName]; !ok {
			return nil, errors.New("unsupported field: " + fieldName)
		}
		if _, dup := seen[fieldName]; dup {
			continue
		}
		seen[fieldName] = struct{}{}
		fields = append(fields, fieldName)
	}
	if len(fields) == 0 {
		return nil, errors.New("fields must include at least one valid field")
	}
	return fields, nil
}
func projectFields(src map[string]any, fields []string) map[string]any {
	out := make(map[string]any, len(fields))
	for _, fieldName := range fields {
		out[fieldName] = src[fieldName]
	}
	return out
}

func containsField(fields []string, target string) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}
