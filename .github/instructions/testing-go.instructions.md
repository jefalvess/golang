---
description: "Use quando escrever, revisar ou refatorar testes Go neste projeto. Cobre table-driven tests, testify, gomock, cobertura mínima, padrões de mock e estrutura de casos de teste. Aplicado automaticamente a todos os arquivos de teste."
applyTo: "**/*_test.go"
---

# Padrões de Testes Go — Comparify

## Filosofia de Testes

Testes documentam o comportamento esperado do sistema. Um teste bem escrito deixa claro:

1. **O que** está sendo testado (função/método)
2. **Em que cenário** (pré-condições e entradas)
3. **Qual resultado é esperado** (saída ou efeito colateral)

Testes não são verificação de implementação — testam **comportamento observável**, não detalhes internos.

## Estrutura de Testes

### Table-Driven Tests (padrão obrigatório)

Todo teste com mais de um cenário deve usar table-driven:

```go
func TestProductService_GetItem(t *testing.T) {
    tests := []struct {
        name       string
        id         string
        fieldsRaw  string
        mockSetup  func(*mockRepository)
        wantFields []string
        wantErr    error
    }{
        {
            name:      "retorna produto com todos os campos quando fields está vazio",
            id:        "phone-1",
            fieldsRaw: "",
            mockSetup: func(m *mockRepository) {
                m.EXPECT().GetByID(gomock.Any(), "phone-1").Return(stubPhone(), nil)
                m.EXPECT().GetSpecificationsByModel(gomock.Any(), gomock.Any(), gomock.Any()).
                    Return(stubSpecs(), nil)
            },
            wantFields: []string{"id", "name", "price"},
            wantErr:    nil,
        },
        {
            name:      "retorna ErrProductNotFound quando produto não existe",
            id:        "nao-existe",
            fieldsRaw: "",
            mockSetup: func(m *mockRepository) {
                m.EXPECT().GetByID(gomock.Any(), "nao-existe").
                    Return(model.Product{}, repository.ErrProductNotFound)
            },
            wantErr: repository.ErrProductNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            repo := NewMockRepository(ctrl)
            tt.mockSetup(repo)

            svc := service.NewProductService(repo)
            got, err := svc.GetItem(context.Background(), tt.id, tt.fieldsRaw)

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }

            require.NoError(t, err)
            for _, field := range tt.wantFields {
                assert.Contains(t, got, field)
            }
        })
    }
}
```

### Nomeação dos testes

Formato: `Test<NomeDaFunção>` ou `Test<Struct>_<Método>`:

```go
func TestGetItem(t *testing.T)                    // handler
func TestHandler_Compare(t *testing.T)            // handler com receiver
func TestProductService_GetItem(t *testing.T)     // service
func TestSQLiteRepository_ListByIDs(t *testing.T) // repository
func TestProduct_FieldMap(t *testing.T)           // model
```

Nomes dos subtestes devem descrever o cenário em português, em letras minúsculas:

```go
t.Run("retorna 404 quando produto não existe", func(t *testing.T) { ... })
t.Run("retorna 400 quando ids está vazio", func(t *testing.T) { ... })
t.Run("retorna itens ordenados conforme os ids fornecidos", func(t *testing.T) { ... })
```

## Assertivas: `require` vs `assert`

| Package | Quando usar |
|---------|------------|
| `require` | Falha fatal — o teste não faz sentido continuar sem essa condição |
| `assert` | Falha não-fatal — continue verificando outras condições |

```go
// require: se err != nil, o restante do teste é inválido
require.NoError(t, err)
require.NotNil(t, got)

// assert: verifica múltiplas propriedades do resultado
assert.Equal(t, "phone-1", got["id"])
assert.Equal(t, "Atlas One", got["name"])
assert.NotContains(t, got, "specifications") // campo não foi solicitado
```

### Verificação de erros

```go
// Para erros sentinela — use errors.Is via require.ErrorIs
require.ErrorIs(t, err, repository.ErrProductNotFound)

// Para erros com mensagem específica
require.EqualError(t, err, "invalid field selection: campo inválido")

// Nunca compare err.Error() como string manualmente
```

## Mocks com gomock

### Geração de mocks

Mocks são gerados automaticamente via `go:generate`. **Nunca** escreva mocks manualmente para as interfaces `Service` e `Repository`:

```go
//go:generate mockgen -source=service.go -destination=service_mock.go -package=service
//go:generate mockgen -source=repository.go -destination=repository_mock.go -package=repository
```

Execute `make mocks` para regenerar após alterar interfaces.

### Padrão de setup de mock

```go
func TestHandler_GetItem(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockSvc := service.NewMockService(ctrl)
    h := handler.NewHandler(mockSvc)

    // Configure exatamente o que o mock deve receber e retornar
    mockSvc.EXPECT().
        GetItem(gomock.Any(), "phone-1", "").
        Return(map[string]any{"id": "phone-1"}, nil).
        Times(1)
}
```

### Matchers

```go
gomock.Any()          // qualquer valor (use com moderação — prefira valores exatos)
gomock.Eq("phone-1") // equivalente a passar o valor direto
gomock.Not(nil)       // qualquer não-nil
```

Prefira valores exatos nos matchers — `gomock.Any()` esconde erros de chamada com argumentos incorretos.

### Quando usar mock local (struct anônima)

Para testes de handler onde o comportamento do mock é simples e não requer verificação de chamadas, uma struct local é aceitável:

```go
type mockService struct {
    getItemFunc func(ctx context.Context, id, fields string) (map[string]any, error)
}

func (m *mockService) GetItem(ctx context.Context, id, fields string) (map[string]any, error) {
    return m.getItemFunc(ctx, id, fields)
}

// Garante que a interface é satisfeita em tempo de compilação
var _ service.Service = (*mockService)(nil)
```

## Testes de Integração com SQLite

Para testar o repositório, use o banco real in-memory:

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })

    // aplica schema e seed de teste
    err = repository.ApplySchema(db)
    require.NoError(t, err)

    return db
}
```

- Use `t.Helper()` em funções auxiliares de setup para que falhas apontem para o teste, não para o helper
- Use `t.Cleanup()` para liberar recursos — evite defer manual em testes com subtests
- Isole os dados de cada teste — não dependa de estado compartilhado entre testes

## Fixtures e Stubs

Crie funções auxiliares para dados de teste reutilizáveis no mesmo pacote:

```go
// stub_test.go (ou no próprio arquivo _test.go)
func stubPhone() model.Product {
    return model.Product{
        ID:    "phone-1",
        Name:  "Atlas One",
        Type:  "celular",
        Model: "One 2026",
        Price: 3299.90,
    }
}

func stubSpecs() map[string]string {
    return map[string]string{
        "batteryCapacity":      "5000 mAh",
        "cameraSpecifications": "50 MP",
        "memory":               "8 GB",
        "storageCapacity":      "256 GB",
        "brand":                "Atlas",
        "operatingSystem":      "Android 16",
    }
}
```

## Testes HTTP (handler)

Use `httptest.NewRecorder()` e `echo.New()` — sem precisar subir servidor real:

```go
func TestHandler_GetItem_Success(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/v1/products/phone-1?fields=id,name", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("phone-1")

    // setup mock e handler
    err := h.GetItem(c)

    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Body.String(), "Atlas One")
}
```

## Cobertura Mínima

Execute `make coverage` para ver a cobertura atual. O projeto deve manter:

| Pacote | Cobertura mínima |
|--------|-----------------|
| `internal/handler` | ≥ 80% |
| `internal/service` | ≥ 85% |
| `internal/repository` | ≥ 75% |
| `internal/model` | ≥ 90% |

Arquivos `_mock.go` são excluídos do relatório de cobertura (`make coverage` já filtra com `grep -v "_mock.go"`).

## Proibições

- **Nunca** adicione lógica de negócio nos testes — se você está testando um if complexo dentro do teste, mova-o para o código de produção
- **Nunca** use `time.Sleep` em testes — use `t.Cleanup`, channels ou waitgroups
- **Nunca** dependa de ordem de execução entre testes — cada teste deve ser independente
- **Nunca** delete um teste existente ao refatorar — atualize-o para refletir o novo comportamento
- **Nunca** use `os.Exit` ou `log.Fatal` em testes — use `t.Fatal` ou `require`
- **Nunca** faça assertions em goroutines não-controladas — use `sync.WaitGroup` ou channels
