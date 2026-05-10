---
description: "Use quando implementar, revisar ou refatorar qualquer endpoint, model, service ou repository da API Comparify. Contém os requisitos funcionais e não-funcionais do desafio transcritos como regras de implementação. Consulte sempre antes de alterar handlers, services, models ou schema."
---

# Requisitos da API Comparify — Fonte de Verdade

> Derivado de `pre-requisitos.md`. Em caso de conflito, o arquivo original prevalece.

## Objetivo da API

Fornecer detalhes de produtos para comparação entre itens. A API deve ser clara, eficiente e seguir boas práticas RESTful.

## Endpoints Obrigatórios

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/v1/products/:id` | Retorna um produto por ID |
| GET | `/v1/products/compare` | Retorna múltiplos produtos para comparação |

### GET `/v1/products/:id`

**Query params opcionais:**

- `fields` — lista de campos separados por vírgula a retornar (ex: `?fields=id,name,price`)

**Resposta de sucesso (200):**

```json
{
  "item": {
    "id": "phone-1",
    "name": "Atlas One",
    "price": 3299.90
  }
}
```

**Respostas de erro:**

| Status | Quando |
|--------|--------|
| 400 | `fields` contém campo inválido |
| 404 | Produto não encontrado |
| 500 | Falha interna inesperada |

### GET `/v1/products/compare`

**Query params obrigatórios:**

- `ids` — lista de IDs separados por vírgula (ex: `?ids=phone-1,phone-2`)

**Query params opcionais:**

- `fields` — campos a retornar (ex: `?fields=id,name,price,specifications`)

**Comportamento:**

- Retorna todos os produtos na **mesma ordem** dos IDs fornecidos
- Se qualquer ID não existir, retorna 404
- Parâmetros desconhecidos na query string resultam em 400

**Resposta de sucesso (200):**

```json
{
  "items": [
    { "id": "phone-1", "name": "Atlas One", "price": 3299.90 },
    { "id": "phone-2", "name": "Nimbus Pro", "price": 4599.00 }
  ],
  "count": 2
}
```

## Modelo de Produto

### Campos base — obrigatórios para todos os tipos

Estes campos devem existir no `model.Product` e estar disponíveis na projeção via `?fields`:

| Campo JSON | Tipo Go | Descrição |
|------------|---------|-----------|
| `id` | `string` | Identificador único |
| `name` | `string` | Nome do produto |
| `imageUrl` | `string` | URL da imagem do produto |
| `description` | `string` | Descrição textual |
| `price` | `float64` | Preço em reais |
| `rating` | `float64` | Avaliação (ex: 0.0 a 5.0) |
| `size` | `string` | Dimensões físicas |
| `weight` | `string` | Peso com unidade |
| `color` | `string` | Cor do produto |
| `type` | `string` | Categoria do produto (ex: "celular", "caixa de som") |
| `model` | `string` | Modelo interno (usado para buscar specs) |

### Campo `specifications` — especialização por tipo

O campo `specifications` é um `map[string]string` que contém atributos específicos do tipo de produto. **Deve ser incluído automaticamente** ao buscar um produto — não é necessário solicitar via `?fields=specifications` explicitamente, mas pode ser excluído se o usuário não o solicitar.

#### Smartphone (`type: "celular"`)

| Chave em `specifications` | Descrição |
|--------------------------|-----------|
| `batteryCapacity` | Capacidade da bateria (ex: "5000 mAh") |
| `cameraSpecifications` | Specs da câmera (ex: "50 MP wide + 12 MP ultrawide") |
| `memory` | RAM (ex: "8 GB") |
| `storageCapacity` | Armazenamento (ex: "256 GB") |
| `brand` | Marca (ex: "Atlas") |
| `modelVersion` | Versão do modelo (ex: "One 2026") |
| `operatingSystem` | Sistema operacional (ex: "Android 16") |

#### Caixa de som (`type: "caixa de som"`)

| Chave | Descrição |
|-------|-----------|
| `brand` | Marca |
| `batteryCapacity` | Duração da bateria (ex: "12 hours") |
| `connectivity` | Conectividade (ex: "Bluetooth 5.3, USB-C") |

#### Geladeira (`type: "geladeira"`)

| Chave | Descrição |
|-------|-----------|
| `brand` | Marca |
| `capacity` | Volume em litros |
| `energyClass` | Classe de eficiência energética |

#### Micro-ondas (`type: "micro-ondas"`)

| Chave | Descrição |
|-------|-----------|
| `brand` | Marca |
| `capacity` | Volume em litros |
| `power` | Potência em watts |

## Projeção de Campos (`?fields`)

O sistema de projeção permite ao cliente ignorar campos não relevantes:

- Se `fields` está **ausente ou vazio**, retorna **todos os campos** do produto incluindo `specifications`
- Se `fields` está **presente**, retorna **apenas** os campos listados
- Se `fields` contém um campo **inválido** (não presente no `FieldMap()`), retorna **400 Bad Request**
- A chave `specifications` pode ser incluída em `fields` para trazer as specs completas

**Exemplos válidos:**

```
GET /v1/products/phone-1?fields=id,name,price
GET /v1/products/compare?ids=phone-1,phone-2&fields=id,name,specifications
GET /v1/products/phone-1                        (sem fields — retorna tudo)
```

**Exemplo inválido:**

```
GET /v1/products/phone-1?fields=id,campoInexistente  → 400 Bad Request
```

## Persistência

- **Mecanismo:** SQLite in-memory (`file:comparify?mode=memory&cache=shared`)
- **Não substitua** por outro banco — é requisito do desafio
- O schema está em `data/schema.sql` (referência) e é aplicado via `internal/repository/schema.go`
- Os dados de seed estão em `data/products.json` e são carregados no `cmd/main.go`
- Especificações de produto ficam em tabelas dedicadas por tipo: `smartphone_specs`, `speaker_specs`, `fridge_specs`, `microwave_specs`
- A tabela `product_type_specs` mapeia `product_type → specs_table` para lookup dinâmico

### Invariantes do banco

- Todo produto tem `type` e `model` — são obrigatórios no schema
- O campo `model` é a chave estrangeira lógica para as tabelas de specs
- Produtos do mesmo modelo compartilham a mesma linha de specs

## Tratamento de Erros — Contrato HTTP

| Situação | Status | Body |
|----------|--------|------|
| Produto não encontrado | 404 | `{"error": "item not found"}` |
| IDs não fornecidos no compare | 400 | `{"error": "ids query parameter is required"}` |
| Campo inválido em `fields` | 400 | `{"error": "invalid field selection: <campo>"}` |
| Parâmetro desconhecido no compare | 400 | `{"error": "unsupported compare query parameter: <param>"}` |
| Erro interno do servidor | 500 | `{"error": "failed to load item"}` ou `"failed to load items"` |

Nunca exponha detalhes internos (query SQL, stack trace, nome de coluna) nas respostas de erro.

## Requisitos Não-Funcionais

### Tratamento de erros

- Erros de domínio como variáveis sentinela (`ErrProductNotFound`, `ErrInvalidFieldSelection`)
- Wrapping com contexto em cada camada
- Mapeamento para HTTP somente no handler

### Logging

- Use o logger Zap centralizado de `pkg/logger`
- Registre início e fim de operações relevantes com campos estruturados
- Nível `Info` para fluxo normal, `Warn` para situações degradadas (ex: specs não encontradas), `Error` para falhas
- Inclua `duration` nas operações de service para observabilidade

### Timeouts

- Timeout de request: 15s (configurado no middleware do servidor)
- Propague o contexto do request para todas as chamadas de I/O

### Makefile

Os seguintes targets devem permanecer funcionando:

| Target | Ação |
|--------|------|
| `make run` | Inicia a API localmente na porta configurada (padrão 8080) |
| `make test` | Executa todos os testes |
| `make build` | Compila a aplicação |
| `make fmt` | Formata o código Go |
| `make vendor` | Atualiza a pasta vendor |
| `make mocks` | Regenera os mocks via `go generate` |
| `make coverage` | Gera relatório de cobertura (excluindo `_mock.go`) |
| `make help` | Lista os targets disponíveis |

### README

O `README.md` deve conter:

1. **Design da API** — arquitetura em camadas, responsabilidades
2. **Endpoints principais** — com exemplos de request/response
3. **Instruções de setup** — pré-requisitos, como rodar, como testar
4. **Decisões arquiteturais** — justificativas das escolhas técnicas

## Checklist de Conformidade

Antes de considerar qualquer feature/refactoring completo, verifique:

- [ ] Todos os campos base do produto estão presentes no `model.Product`
- [ ] Campos de smartphone estão na tabela `smartphone_specs` e mapeados em `specifications`
- [ ] `?fields` aceita qualquer subconjunto válido de campos e retorna 400 para inválidos
- [ ] `GET /compare` aceita apenas `ids` e `fields` como query params
- [ ] Erros retornam os status HTTP corretos conforme a tabela acima
- [ ] `make test` passa sem erros
- [ ] Nenhuma lógica de negócio foi adicionada ao handler
- [ ] Nenhum acesso direto ao banco foi feito fora do `repository/`
- [ ] O campo `model` no produto (usado internamente para specs) não é confundido com `modelVersion` das specs do smartphone
