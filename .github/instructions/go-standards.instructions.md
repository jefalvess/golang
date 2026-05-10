---
description: "Use quando escrever, revisar ou refatorar código Go neste projeto. Cobre SOLID, Clean Code, convenções de nomeação, tratamento de erros, estrutura de pacotes e padrões de interface. Aplicado automaticamente a todos os arquivos Go."
applyTo: "**/*.go"
---

# Padrões de Código Go — Comparify

## Estrutura de Pacotes e Responsabilidades

Cada pacote tem uma responsabilidade única e não deve extrapolá-la:

| Pacote | Responsabilidade | Proibido |
|--------|-----------------|---------|
| `internal/handler` | Decodificar request, chamar service, encodar response | Lógica de negócio, acesso a dados |
| `internal/service` | Regras de negócio, orquestração | Acesso direto ao banco, detalhes HTTP |
| `internal/repository` | Acesso ao SQLite, mapeamento de linhas | Regras de negócio, lógica de projeção |
| `internal/model` | Definição de structs e helpers de dados | Dependências externas |
| `internal/server` | Roteamento, middlewares, lifecycle do servidor | Lógica de negócio |
| `pkg/logger` | Logging Zap centralizado | Estado mutável global |
| `pkg/utils` | Funções utilitárias puras sem estado | Dependências de domínio |

## Princípios SOLID

### Single Responsibility Principle (SRP)

- Cada função tem **uma** razão para mudar
- Se uma função lê request, transforma dados E escreve response, divida-a
- Funções de parsing, validação e formatação devem ser separadas

```go
// Correto: separação clara de responsabilidades
func (h *Handler) Compare(c echo.Context) error {
    ids, err := readCompareSelection(c)  // parsing isolado
    if err != nil {
        return writeError(c, http.StatusBadRequest, err.Error())
    }
    items, err := h.productService.Compare(c.Request().Context(), ids, c.QueryParam("fields"))
    // ...
}

// Errado: misturar parsing de query, validação de negócio e acesso a dados no handler
```

### Open/Closed Principle (OCP)

- Para adicionar suporte a novo tipo de produto: adicione nova tabela de specs + entrada em `product_type_specs`
- **Não** adicione `if product.Type == "novo_tipo"` no código existente
- Use o mecanismo de lookup dinâmico de tabela já estabelecido no `sqlite_repository.go`

### Liskov Substitution Principle (LSP)

- Qualquer implementação de `Repository` deve se comportar conforme a interface define
- `ErrProductNotFound` deve ser retornado por **qualquer** implementação quando o produto não existe
- Mocks devem respeitar exatamente o contrato das interfaces reais

### Interface Segregation Principle (ISP)

- Prefira interfaces pequenas e focadas
- Se um consumidor usa apenas `GetByID`, não force-o a depender de uma interface com 10 métodos
- Ao criar novas interfaces, inclua apenas os métodos que aquele consumidor precisa

### Dependency Inversion Principle (DIP)

- Dependências são sempre injetadas via construtor — nunca instanciadas internamente
- Construtores recebem interfaces, não implementações concretas:

```go
// Correto
func NewProductService(repo repository.Repository) *ProductService { ... }

// Errado
func NewProductService() *ProductService {
    repo := &SQLiteRepository{} // acoplamento concreto
}
```

## Convenções de Nomeação Go

### Funções e métodos

- Use nomes que descrevam a ação: `parseFields`, `selectFields`, `readCompareSelection`
- Prefira verbos: `Get`, `List`, `Parse`, `Select`, `Build`, `Resolve`
- Evite sufixos genéricos como `Manager`, `Helper`, `Util`, `Data`
- Funções exportadas: `PascalCase`; funções internas: `camelCase`

### Variáveis

- Nomes curtos e contextuais em escopos pequenos: `id`, `ids`, `err`, `ctx`
- Nomes descritivos em escopos maiores: `productRepository`, `specsByModelAndType`
- Evite abreviações não-óbvias: prefira `product` a `p` em funções longas

### Erros sentinela

```go
// Declare no pacote onde o erro é originado
var ErrProductNotFound = errors.New("product not found")
var ErrInvalidFieldSelection = errors.New("invalid field selection")
```

### Constantes

```go
// Prefira constantes tipadas e agrupadas
const (
    apiVersionPrefix = "/v1"
    requestTimeout   = 15 * time.Second
)
```

## Tratamento de Erros

### Regras obrigatórias

1. **Sempre** verifique erros retornados — nunca use `_` para erros de operações relevantes
2. **Sempre** faça wrapping com contexto ao propagar erros entre camadas:

```go
// Correto: preserva a cadeia de erros
return nil, fmt.Errorf("GetItem: fetching specs for model %s: %w", product.Model, err)

// Errado: perde contexto
return nil, err
```

3. **Compare** erros com `errors.Is()` — nunca compare strings de erro
4. **Mapeie** erros de domínio para status HTTP apenas no handler:

```go
// No handler — único lugar onde erros viram status HTTP
switch {
case errors.Is(err, repository.ErrProductNotFound):
    return writeError(c, http.StatusNotFound, "item not found")
case errors.Is(err, service.ErrInvalidFieldSelection):
    return writeError(c, http.StatusBadRequest, err.Error())
default:
    return writeError(c, http.StatusInternalServerError, "failed to load item")
}
```

5. **Nunca** exponha mensagens internas ao cliente (stack trace, query SQL, nome de coluna)

### Proibido

- `panic()` fora de `cmd/main.go` ou `init()`
- Ignorar erros retornados por funções críticas
- Retornar `nil, nil` quando um resultado vazio deveria ser `nil, ErrNotFound`

## Comentários e Documentação

### Quando comentar

- Comentários explicam **intenção** (o porquê), não o **o quê** (o código já diz)
- Comente apenas quando a lógica não é imediatamente óbvia:

```go
// Correto: explica a intenção não-óbvia
// O IN do SQLite não preserva a ordem de entrada; reordenamos para refletir o compare pedido.
orderedProducts = append(orderedProducts, product)

// Errado: redundante com o código
// Incrementa o contador
count++
```

- Funções exportadas **devem** ter godoc começando com o nome da função:

```go
// FieldMap define o conjunto de campos públicos que pode ser projetado pela API.
func (p Product) FieldMap() map[string]any { ... }
```

### Diretivas `go:generate`

Sempre mantenha as diretivas no topo do arquivo de interface:

```go
//go:generate mockgen -source=service.go -destination=service_mock.go -package=service
```

## Estruturas de Controle

### Early Return

Prefira retorno antecipado para reduzir aninhamento:

```go
// Correto: early return
func (h *Handler) GetItem(c echo.Context) error {
    itemID := c.Param("id")
    if itemID == "" {
        return writeError(c, http.StatusNotFound, "item not found")
    }
    // fluxo principal sem aninhamento extra
}

// Errado: else desnecessário após return
if itemID == "" {
    return writeError(...)
} else {
    // continua aqui
}
```

### Literais de mapa e slice

```go
// Inicialize com capacidade conhecida quando possível
items := make([]map[string]any, 0, len(products))
result := make(map[string]string, len(specs))
```

## Contexto (`context.Context`)

- Todo método que acessa I/O (banco, rede) deve receber `ctx context.Context` como **primeiro parâmetro**
- Nunca armazene `context.Context` em structs
- Propague o contexto do request: `c.Request().Context()`
- Use `ctx` para cancelamento e timeout — o middleware de timeout já configura o deadline

## Imports

Organize em três grupos separados por linha em branco:

```go
import (
    // stdlib
    "context"
    "errors"
    "fmt"

    // pacotes internos
    "comparify/internal/model"
    "comparify/pkg/logger"

    // dependências externas
    "github.com/labstack/echo/v4"
    "go.uber.org/zap"
)
```

## Formatação

- Execute `make fmt` antes de qualquer commit
- O projeto usa `gofmt` padrão — sem configuração adicional de linter necessária
- Tamanho de linha: sem limite rígido, mas prefira legibilidade a linhas muito longas
