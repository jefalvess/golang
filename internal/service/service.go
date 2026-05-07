//go:generate mockgen -source=service.go -destination=service_mock.go -package=service

package service

import "context"

// Service define o contrato de regras de negócio exposto ao handler HTTP.
// Permite desacoplar o handler da implementação concreta e facilita testes.
type Service interface {
	// GetItem retorna um produto por ID, projetando apenas os campos pedidos.
	GetItem(ctx context.Context, id string, fieldsRaw string) (map[string]any, error)
	// Compare retorna múltiplos produtos por ids, projetando apenas os campos pedidos.
	Compare(ctx context.Context, ids []string, fieldsRaw string) ([]map[string]any, error)
}
