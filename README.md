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

## Simulação com massa de dados via Scripts

<img width="967" height="1039" alt="20260818 113304" src="https://github.com/user-attachments/assets/28e256d7-54af-4845-aebf-27a11d3260d8" />
<img width="1920" height="1040" alt="20260818 113308" src="https://github.com/user-attachments/assets/b88e4124-c33e-422a-9439-87f06c8af538" />
<img width="967" height="1039" alt="20260818 113352" src="https://github.com/user-attachments/assets/7ae1b09b-9f3a-4e20-8b4d-989b7150ffdf" />
<img width="967" height="1039" alt="20260818 113403" src="https://github.com/user-attachments/assets/87835688-3bf6-4c12-b2ee-f99e34d4a07a" />
<img width="967" height="1039" alt="20260818 113407" src="https://github.com/user-attachments/assets/e59eabaf-5db7-41f7-811e-a0807edf678d" />
<img width="967" height="1039" alt="20260818 113410" src="https://github.com/user-attachments/assets/33a4f6c1-c2f0-4aea-9f39-e2159b2065f6" />
<img width="967" height="1039" alt="20260818 113435" src="https://github.com/user-attachments/assets/c22756d1-7c64-4947-a22e-2599c86ffbf7" />
<img width="967" height="1039" alt="20260818 113439" src="https://github.com/user-attachments/assets/91694b24-a5a5-46c2-a827-aafa2a2a320e" />
<img width="967" height="1039" alt="20260818 113444" src="https://github.com/user-attachments/assets/fb7fd8ee-078d-45dd-a153-056149638c91" />
<img width="967" height="1039" alt="20260818 113449" src="https://github.com/user-attachments/assets/49d816e3-36a0-432f-9739-5d29149b64b7" />
<img width="1920" height="1040" alt="20260818 113512" src="https://github.com/user-attachments/assets/6d02ce44-3f4d-4d12-801f-347e98390b8c" />
<img width="967" height="1039" alt="20260818 113535" src="https://github.com/user-attachments/assets/1e9d7c5c-e9d3-4756-8041-85a25a4cf1fb" />
<img width="967" height="1039" alt="20260818 113543" src="https://github.com/user-attachments/assets/bf487fdf-c93f-4055-b57e-7d3293d43471" />
<img width="967" height="1039" alt="20260818 113550" src="https://github.com/user-attachments/assets/0ecc1c15-4e07-40e3-8aa6-48b217514f02" />
<img width="967" height="1039" alt="20260818 113556" src="https://github.com/user-attachments/assets/d29ed299-56d3-4bd4-98cb-40c75f7663ac" />
<img width="1920" height="1040" alt="20260818 113616" src="https://github.com/user-attachments/assets/85c357a5-2579-4abc-9084-43da6f67ced6" />
<img width="967" height="1039" alt="20260818 113636" src="https://github.com/user-attachments/assets/c52119e0-017f-48dd-856f-7ae40cd29b43" />
<img width="967" height="1039" alt="20260818 113646" src="https://github.com/user-attachments/assets/71c8e4d7-ba08-499d-8807-7a7a49142fd6" />
<img width="967" height="1039" alt="20260818 113651" src="https://github.com/user-attachments/assets/aa67d0a6-a6e9-41fa-b1ae-9e8dc5131db3" />
<img width="967" height="1039" alt="20260818 113658" src="https://github.com/user-attachments/assets/c20be6e5-9f56-46bb-94e1-54ad6c94d2a2" />
<img width="1920" height="1040" alt="20260818 113721" src="https://github.com/user-attachments/assets/d0d90944-8e41-4bce-9a4f-76dd5d560a7d" />



> Documento detalhado (ciclos de vida Angular, RxJS, bibliotecas, tratamento de erros no backend, frameworks Go etc.): **[DETALHAMENTO_TECNICO.md](./DETALHAMENTO_TECNICO.md)**
