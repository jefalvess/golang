//go:generate mockgen -source=service.go -destination=service_mock.go -package=service

package service

import "context"

// Service define o contrato de regras de negócio exposto ao handler HTTP.
// Permite desacoplar o handler da implementação concreta e facilita testes.
type Service interface {
	// ListItems retorna todos os produtos cadastrados, projetando apenas os campos pedidos.
	ListItems(ctx context.Context) ([]map[string]any, error)
	// AvailableFields retorna os campos de primeiro nível aceitos em ?fields=.
	AvailableFields() []string
	// Compare retorna múltiplos produtos por ids, projetando apenas os campos pedidos.
	Compare(ctx context.Context, ids []string, fieldsRaw string) ([]map[string]any, error)
}
