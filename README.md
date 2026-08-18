# Korp_Teste_Nogueira

Sistema de emissão de **Notas Fiscais** — desafio técnico Korp.

Aplicação full-stack com frontend **Angular**, backend em **Golang** (arquitetura de **microsserviços**) e banco de dados **PostgreSQL**:

- **Serviço de Estoque** (`:8081`) — cadastro de produtos e controle de saldos
- **Serviço de Faturamento** (`:8082`) — gestão e impressão de notas fiscais
- **Frontend Angular** (`:4200`) — interface web com login JWT

---

## Funcionalidades

| Módulo | Descrição |
|---|---|
| **Produtos** | Cadastro com código, descrição e saldo em estoque; edição |
| **Notas Fiscais** | Criação com numeração sequencial automática, status inicial **Aberta** e múltiplos produtos/quantidades |
| **Impressão** | Botão visível apenas para notas **Abertas**; indicador de processamento; ao concluir a nota passa a **Fechada** e o saldo dos produtos é **baixado** automaticamente |
| **Logs** | Aba que agrega, em tempo real, o histórico de tudo que foi processado pelos dois serviços (login, produtos criados/atualizados, baixa de estoque, notas criadas, impressões e erros) com atualização automática a cada 3s |
| **Controle de serviços** | Botões na toolbar para **parar e iniciar** cada microsserviço em tempo real (via supervisor `control` em `:8080`), com badges de saúde refletindo up/down |

### Recursos opcionais implementados

- **Concorrência** — a baixa de estoque usa `UPDATE ... WHERE stock_quantity >= qtd` em transação atômica; duas notas disputando saldo 1: apenas uma vence (a outra recebe 409). Veja `scripts/simulate-concurrency.sh`.
- **Tratamento de falhas** — Faturamento chama Estoque com **retry + backoff exponencial** e **circuit breaker**; com o Estoque fora do ar o print retorna **503** amigável e a nota permanece Aberta para nova tentativa. Veja `scripts/simulate-failure.sh`.
- **Idempotência** — o `POST /print` aceita o header `Idempotency-Key`; repetições da mesma operação retornam o resultado salvo sem efeitos colaterais.
- **Concorrência** — a baixa de estoque usa `UPDATE ... WHERE stock_quantity >= qtd` em transação atômica; duas notas disputando saldo 1: apenas uma vence (a outra recebe 409). Veja `scripts/simulate-concurrency.sh`.
- **Tratamento de falhas** — Faturamento chama Estoque com **retry + backoff exponencial** e **circuit breaker**; com o Estoque fora do ar o print retorna **503** amigável e a nota permanece Aberta para nova tentativa. Veja `scripts/simulate-failure.sh`.

---

## Arquitetura

```
┌──────────────┐    HTTP (JWT)    ┌──────────────────┐   HTTP (X-Internal-Token)
│   Angular    │ ───────────────▶ │  Faturamento :8082│ ─────────────────────────▶ ┌────────────┐
│    :4200     │                  │ (notas fiscais)   │        consume stock       │ Estoque :8081│
└──────────────┘ ───────────────▶ └──────────────────┘                            └────────────┘
        │ HTTP (JWT)                  (login JWT)                                       (produtos)
        └────────────────────────────────────────────────────────────────────────────────┘
                                   PostgreSQL 14
                     korp_estoque (products) · korp_faturamento (invoices)
```

- Comunicação Faturamento → Estoque feita **server-to-server** com token interno (`X-Internal-Token`), isolando os domínios.
- Cada serviço possui **banco de dados próprio**, respeitando o isolamento de microsserviços.

---

## Stack

| Camada | Tecnologia |
|---|---|
| Frontend | Angular 22 (standalone), Angular Material, RxJS |
| Backend | Go 1.26, `chi` (roteador), `pgx` (driver PostgreSQL) |
| Banco | PostgreSQL 14 |
| Autenticação | JWT (HS256), credenciais fixas de demonstração |
| Gerenciamento de dependências Go | Go Modules (`go.mod`) |

---

## Como executar

### Pré-requisitos

- Go 1.26+
- Node.js 20+ / npm
- PostgreSQL 14+ rodando localmente

### 1. Banco de dados

```bash
# Cria o usuário `korp` e os bancos korp_estoque / korp_faturamento (idempotente)
bash scripts/setup-db.sh
```

### 2. Backend (microsserviços)

```bash
bash scripts/run-all.sh          # sobe Estoque (:8081) e Faturamento (:8082)
# opcional: incluir o frontend  -> bash scripts/run-all.sh --frontend
bash scripts/stop-all.sh         # derruba tudo
```

As migrations são aplicadas automaticamente no boot de cada serviço.

### 3. Frontend

```bash
cd frontend/invoice-app
npm install
npx ng serve --port 4200        # proxy /api/faturamento e /api/estoque configurado
```

Acesse **http://localhost:4200** — login de demonstração: **admin** / **admin123**.

### Variáveis de ambiente (opcionais)

Ambos os serviços possuem defaults prontos para desenvolvimento local.

| Variável | Serviço | Default |
|---|---|---|
| `DATABASE_URL` | ambos | `postgres://korp:korp_dev_2026@localhost:5432/korp_*` |
| `JWT_SECRET` | ambos | `korp-jwt-secret-dev-change-me` |
| `INTERNAL_TOKEN` | ambos | `korp-internal-token-dev` |
| `ESTOQUE_URL` | faturamento | `http://localhost:8081` |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | faturamento | `admin` / `admin123` |
| `ESTOQUE_BASE_URL` / `ESTOQUE_API_KEY` / `ESTOQUE_MODEL` | estoque | — (descrição gerada localmente) |

---

## Testes e simulações

```bash
bash scripts/e2e-smoke.sh               # 14 asserções de ponta a ponta
bash scripts/simulate-concurrency.sh    # saldo 1 disputado por 2 notas simultâneas
bash scripts/simulate-failure.sh        # Estoque fora do ar → 503 → recuperação
```

---

## API resumida

### Estoque (`:8081`)
| Método | Rota | Descrição |
|---|---|---|
| POST | `/api/products` | Cria produto |
| GET | `/api/products` | Lista produtos |
| GET | `/api/products/{code}` | Busca produto |
| PUT | `/api/products/{code}` | Atualiza produto |
| POST | `/api/products/ai-description` | Sugere descrição do produto |
| POST | `/api/products/consume` | Baixa estoque (transação atômica) |
| GET | `/api/logs` | Histórico de processamento do Estoque |
| GET | `/health` | Health check |

### Faturamento (`:8082`)
| Método | Rota | Descrição |
|---|---|---|
| POST | `/api/auth/login` | Login → JWT |
| POST | `/api/invoices` | Cria nota (numeração sequencial) |
| GET | `/api/invoices` | Lista notas |
| GET | `/api/invoices/{id}` | Busca nota |
| POST | `/api/invoices/{id}/print` | Imprime/fecha nota (idempotente via `Idempotency-Key`) |
| GET | `/api/logs` | Histórico de processamento do Faturamento |
| GET | `/health` | Health check |

### Control (`:8080`)
| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/status` | Estado atual dos dois serviços |
| POST | `/api/estoque/start` \| `/api/estoque/stop` | Inicia/para o Estoque |
| POST | `/api/faturamento/start` \| `/api/faturamento/stop` | Inicia/para o Faturamento |
| GET | `/health` | Health check |

---

## Estrutura do projeto

```
backend/
├── estoque/          # Microsserviço de Estoque (Go)
├── faturamento/      # Microsserviço de Faturamento (Go)
└── control/          # Supervisor que inicia/para os serviços (:8080)
frontend/invoice-app/ # Aplicação Angular
scripts/              # setup-db, run-all, stop-all, e2e-smoke, simulações
DETALHAMENTO_TECNICO.md  # Detalhamento técnico exigido pelo desafio
```

> Documento detalhado (ciclos de vida Angular, RxJS, bibliotecas, tratamento de erros no backend, frameworks Go etc.): **[DETALHAMENTO_TECNICO.md](./DETALHAMENTO_TECNICO.md)**