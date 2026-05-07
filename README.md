# Diagrama de Arquitetura

```mermaid
graph TD
  A[Cliente] -->|HTTP| B[Handler]
  B --> C[Service (interface)]
  C --> D[Repository (interface)]
  D --> E[(SQLite em memória)]
  E --> T1[products]
  E --> T2[smartphone_specs]
  E --> T3[fridge_specs]
  E --> T4[microwave_specs]
  E --> T5[speaker_specs]
  C --> F[Utils]
  B --> G[Logger zap]
  C --> G
  D --> G
  B -->|Responde JSON| A
  J[products.json] -.seed.-> D
```

# Item Comparison API

API backend em Go para comparação de produtos, seguindo boas práticas de arquitetura, logging estruturado, documentação e testes. Atende requisitos de entrevista técnica e simula persistência com SQLite em memória e seed JSON.

---

## Visão Geral

Esta API permite consultar detalhes de produtos e realizar comparações flexíveis, retornando apenas os campos desejados pelo cliente. O modelo de produto cobre atributos essenciais e permite extensões para especificações especializadas (ex: smartphones).

**Principais características:**
- Seleção explícita de itens por `ids`
- Projeção de campos via parâmetro `fields`
- Logging estruturado com níveis (info, warn, error) usando zap
- Tratamento consistente de erros
- Arquitetura modular (handler, service.Service, repository.Repository, utils)
- Contratos de interface claros para service e repository
- Documentação e exemplos completos

---


## Comparação de Itens

Com base no enunciado, a comparação é feita entre itens específicos. O endpoint de comparação aceita somente estes parâmetros:

| Parâmetro | Tipo | Exemplo | Observação |
|---|---|---|---|
| `ids` | seleção explícita | `ids=phone-1,phone-2` | obrigatório para o endpoint de comparação |
| `fields` | projeção | `fields=id,name,price` | não filtra; apenas limita os campos retornados |

Regras de uso:

- `ids` é obrigatório no endpoint de comparação.
- os ids devem ser enviados separados por vírgula.
- `fields` é opcional.
- Qualquer outro parâmetro de query no endpoint de compare retorna erro `400`.

```bash
curl "http://localhost:8080/v1/products/compare?ids=phone-1,phone-2&fields=id,name,price,specifications"
```

Esse exemplo retorna exatamente os itens informados em `ids`, com apenas os campos pedidos em `fields`.

---

## Endpoints

- `GET /v1/products/{id}`: retorna um produto por ID. Use `fields=...` para limitar os campos retornados.
- `GET /v1/products/compare?ids=...&fields=...`: retorna produtos específicos para comparação.

**Exemplo de erro:**

```json
{
  "error": {
    "message": "item not found",
    "status": 404
  }
}
```

**Exemplo de erro por seleção inválida no compare:**

```json
{
  "error": {
    "message": "ids query parameter is required",
    "status": 400
  }
}
```


## Modelo de Produto

Campos essenciais (tabela `products`):

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | TEXT (PK) | Identificador único do produto |
| `name` | TEXT | Nome do produto |
| `image_url` | TEXT | URL da imagem |
| `description` | TEXT | Descrição |
| `price` | REAL | Preço |
| `rating` | REAL | Avaliação |
| `size` | TEXT | Dimensões |
| `weight` | TEXT | Peso |
| `color` | TEXT | Cor |
| `type` | TEXT | Tipo (celular, geladeira, etc.) |
| `model` | TEXT | Modelo comercial do produto |

---

## Estrutura do Banco de Dados

O banco SQLite em memória é composto por uma tabela principal, uma tabela de metadados por tipo e quatro tabelas de especificações especializadas. As especificações são reutilizadas por `model`, evitando duplicação por `product_id`.

### `product_type_specs`

| Campo | Tipo | Descrição |
|---|---|---|
| `product_type` | TEXT (PK) | Tipo do produto |
| `specs_table` | TEXT | Nome da tabela de especificações daquele tipo |

### `smartphone_specs`

| Campo | Tipo | Descrição |
|---|---|---|
| `model` | TEXT (PK) | Modelo comercial do produto |
| `battery_capacity` | TEXT | Capacidade da bateria |
| `camera_specs` | TEXT | Especificações de câmera |
| `memory` | TEXT | Memória RAM |
| `storage_capacity` | TEXT | Armazenamento |
| `brand` | TEXT | Marca |
| `operating_system` | TEXT | Sistema operacional |

### `fridge_specs`

| Campo | Tipo | Descrição |
|---|---|---|
| `model` | TEXT (PK) | Modelo comercial do produto |
| `capacity` | TEXT | Capacidade em litros |
| `energy_class` | TEXT | Classificação energética |
| `brand` | TEXT | Marca |

### `microwave_specs`

| Campo | Tipo | Descrição |
|---|---|---|
| `model` | TEXT (PK) | Modelo comercial do produto |
| `capacity` | TEXT | Capacidade em litros |
| `power` | TEXT | Potência em Watts |
| `brand` | TEXT | Marca |

### `speaker_specs`

| Campo | Tipo | Descrição |
|---|---|---|
| `model` | TEXT (PK) | Modelo comercial do produto |
| `battery_capacity` | TEXT | Autonomia da bateria |
| `connectivity` | TEXT | Conectividade (Bluetooth, etc.) |
| `brand` | TEXT | Marca |

---

## Decisões Arquiteturais

- Go 1.22 e Echo para roteamento HTTP
- Persistência simulada: SQLite em memória, seed via JSON
- Logging estruturado com zap (níveis info, warn, error)
- Repository pattern para desacoplamento
- Service layer para lógica de negócio/testabilidade
- Utilitários centralizados (ex: splitAndTrim)
- Testes unitários cobrindo lógica principal
- Documentação e exemplos completos


## Setup e Execução

Pré-requisitos: Go 1.22+

> **Atenção:** este projeto usa `go-sqlite3`, que requer CGO. Sempre execute com `CGO_ENABLED=1`.

```bash
go test ./...
CGO_ENABLED=1 go run ./cmd
```

Porta padrão: `8080`. Para alterar:

```bash
CGO_ENABLED=1 PORT=8080 go run ./cmd
```


## Exemplos de Requisições

```bash

# Buscar um produto específico
curl "http://localhost:8080/v1/products/phone-1"


# Comparar produtos específicos por ids
curl "http://localhost:8080/v1/products/compare?ids=phone-1,phone-2&fields=id,name,price,specifications"


# Exemplo com projeção mínima de campos
curl "http://localhost:8080/v1/products/compare?ids=phone-1,phone-2&fields=name,price"


# Exemplo inválido: ids é obrigatório no compare
curl "http://localhost:8080/v1/products/compare?fields=name,price"

# Exemplo inválido: parâmetro não suportado (apenas 'ids' e 'fields' são aceitos)
curl "http://localhost:8080/v1/products/compare?type=celular&fields=name,price" # retorna erro 400
```

---

## Testes

Execute todos os testes unitários:

```bash
go test ./...
```

---

## Observações

- Logging estruturado (zap) já configurado, com níveis e campos para fácil integração com sistemas de monitoramento.
- O projeto pode ser facilmente estendido para outros tipos de produtos.
- Todos os requisitos funcionais e não-funcionais do desafio estão cobertos.