# Detalhamento Técnico — Sistema de Emissão de Notas Fiscais

> Desafio técnico Korp · Projeto **Korp_Teste_Nogueira**

Este documento responde, item a item, os pontos técnicos exigidos na especificação, além de descrever a arquitetura e as decisões de implementação.

---

## 1. Visão geral da solução

| Componente | Tecnologia | Porta | Responsabilidade |
|---|---|---|---|
| Frontend | Angular 22 (standalone components) | 4200 | Interface web das telas |
| Microsserviço 1 — **Estoque** | Go 1.26 + chi + pgx | 8081 | Produtos e saldos; baixa de estoque |
| Microsserviço 2 — **Faturamento** | Go 1.26 + chi + pgx | 8082 | Login JWT; notas fiscais; impressão |
| Supervisor de serviços | Go 1.26 (stdlib) | 8080 | Inicia/para os microsserviços sob demanda (botões da toolbar) |
| Banco de dados | PostgreSQL 14 | 5432 | Persistência real (`korp_estoque`, `korp_faturamento`) |

Regras de negócio implementadas:

- **Produtos**: código, descrição e saldo (obrigatórios).
- **Notas fiscais**: numeração sequencial automática, status `OPEN` (Aberta) na criação, múltiplos produtos com quantidades.
- **Impressão**: apenas notas **Abertas** podem ser impressas; ao imprimir a nota passa a **Fechada** e o saldo dos produtos é **baixado** (ex.: saldo 10, nota usa 2 → novo saldo 8). Exibe indicador de processamento durante a operação.

---

## 2. Ciclos de vida do Angular utilizados

Foram utilizados os seguintes lifecycle hooks, todos demonstrados no código:

| Hook | Onde | Uso |
|---|---|---|
| `ngOnInit` | `LoginComponent`, `ShellComponent`, `ProductsComponent`, `InvoicesComponent`, `ProductDialogComponent`, `InvoiceDialogComponent` | Carregar dados iniciais, redirecionar usuário autenticado, criar a primeira linha de itens do diálogo de nota |
| `ngOnDestroy` | `ShellComponent`, `ProductsComponent`, `InvoicesComponent` | Emitir o `Subject` combinado com `takeUntil` para cancelar todas as subscrições e evitar *memory leaks* |
| `ngOnChanges` | — (não utilizado) | — |

> Observação: os componentes foram construídos com a nova API de **signals/standalone** do Angular 22, mantendo o padrão de um único arquivo por componente (`*.ts` + `*.html` + `*.scss`). `AfterViewInit` não foi necessário porque as interações com o DOM são feitas pelo Material.

### Detecção de mudança (zoneless)

O Angular 22 é **zoneless por padrão** (desde a v21 não carrega `zone.js`). Nesse modo, a detecção de mudança só é agendada quando: um *signal* lido no template muda, um `AsyncPipe` emite, um evento de template dispara, ou quando `ChangeDetectorRef.markForCheck()` é chamado — **uma subscrição RxJS que apenas atribui uma propriedade comum NÃO dispara re-render automático**.

Por isso, todos os componentes que atualizam estado a partir de operações assíncronas (HTTP, `interval`, `afterClosed` de diálogos) chamam `cdr.markForCheck()` logo após cada atualização de estado dentro das subscrições (`ProductsComponent`, `InvoicesComponent`, `LogsComponent`, `ShellComponent`, `LoginComponent`, `ProductDialogComponent`, `InvoiceDialogComponent`, `ProductDetailComponent`, `InvoiceDetailComponent`). Essa é a prática recomendada para aplicações zoneless que ainda usam RxJS clássico.

---

## 3. Uso da biblioteca RxJS

Sim, RxJS foi usado de forma intensiva. Principais operadores e onde aparecem:

| Recurso RxJS | Onde | Finalidade |
|---|---|---|
| `BehaviorSubject` | `AuthService` (`isAuthenticated$`) | Estado reativo de autenticação; expõe o estado via `Observable` |
| `Observable` / `HttpClient` | Todos os serviços | Assinatura de requisições HTTP |
| `tap` | `AuthService.login` | Efeito colateral ao receber o token (persistir e atualizar estado) |
| `catchError` / `of` | `HealthService` | Converter falha de health check em status `down` amigável em vez de lançar erro |
| `switchMap` | `ShellComponent` | Combinar `interval(5000)` com o health check, cancelando chamadas anteriores (polling de status dos serviços) |
| `takeUntil` | Componentes | Cancelar subscrições no `ngOnDestroy` |
| `forkJoin` | `InvoicesComponent` | Carregar produtos e notas em paralelo na mesma tela |
| `interval` | `ShellComponent` | Polling dos badges de status dos microsserviços a cada 5s |

O JWT é anexado automaticamente a toda requisição por um **HttpInterceptor** (`authInterceptor`) funcional do Angular.

---

## 4. Outras bibliotecas utilizadas (e finalidade)

### Frontend
| Biblioteca | Finalidade |
|---|---|
| `@angular/material` | Componentes visuais (ver seção 5) |
| `rxjs` | Programação reativa (ver seção 3) |

### Backend (Go)
| Biblioteca | Finalidade |
|---|---|
| `github.com/go-chi/chi/v5` | Roteador HTTP leve e idiomaticamente Go |
| `github.com/go-chi/cors` | Middleware CORS para o frontend Angular |
| `github.com/jackc/pgx/v5` (`pgxpool`) | Driver PostgreSQL com *connection pooling* e suporte a transações |
| `github.com/golang-jwt/jwt/v5` | Emissão e validação de tokens JWT (HS256) |

---

## 5. Bibliotecas de componentes visuais

Utilizada a **Angular Material** (Material Design 3):

- `MatTableModule` — tabelas de produtos e notas fiscais
- `MatDialogModule` — diálogos de cadastro de produto e criação de nota
- `MatFormFieldModule`, `MatInputModule`, `MatSelectModule` — formulários (inclusive o select de produtos com saldo disponível)
- `MatButtonModule`, `MatIconModule`, `MatTooltipModule` — ações (botão **Imprimir** com ícone, tooltips)
- `MatProgressSpinnerModule`, `MatProgressBarModule` — indicador de processamento na impressão e carregamento
- `MatChipsModule` — chips de status (Aberta/Fechada) e do saldo em estoque
- `MatSnackBarModule` — feedback de sucesso/erro ao usuário
- `MatCardModule`, `MatToolbarModule`, `MatTabsModule` — layout (login, barra superior com badges de saúde dos serviços, navegação por abas)

---

## 6. Gerenciamento de dependências no Golang

Utilizado **Go Modules** (arquivo `go.mod`), o gerenciador oficial de dependências do Go:

- Cada microsserviço tem seu próprio módulo (`korp/estoque` e `korp/faturamento`).
- `go mod tidy` resolve e registra as dependências diretas e transitivas em `go.sum`.
- As versões são fixadas no `go.mod`, garantindo builds reproduzíveis.
- Não foi utilizado Docker Compose; os serviços rodam como processos independentes (dois bancos PostgreSQL distintos reforçam o isolamento entre os microsserviços).

---

## 7. Frameworks utilizados no backend (Golang)

- **chi** (`go-chi/chi/v5`) — framework web minimalista e compatível com o `net/http` padrão, usado para roteamento, middlewares (request ID, logger, recover, timeout) e organização de rotas em grupos (`r.Route("/api", ...)`).
- **pgx v5** — não é um framework de ORM, mas a biblioteca de driver/ORM-leve padrão para PostgreSQL, usada diretamente com SQL e transações.
- **Não foi utilizado ORM** (como GORM); optou-se por SQL explícito para demonstrar controle fino sobre transações e concorrência.

---

## 8. Tratamento de erros e exceções no backend

Estratégia aplicada nos dois serviços:

1. **Padronização de respostas** — toda falha retorna JSON `{"error": "mensagem"}`, com código HTTP adequado (400 validação, 401 auth, 404 não encontrado, 409 conflito/estoque, 503 indisponível).
2. **Middleware de recuperação** — `chimw.Recoverer` (do chi) captura panics e devolve 500 sem derrubar o processo.
3. **Validação de entrada** — campos obrigatórios, quantidade > 0, código duplicado (mapeado de erro `23505` do PostgreSQL para 409).
4. **Erros de domínio como sentinelas** — `ErrProductNotFound`, `ErrInsufficientStock`, `ErrInvoiceClosed` etc., verificados com `errors.Is`.
5. **Retry + backoff exponencial + circuit breaker** (no Faturamento) — a chamada ao Estoque:
   - tenta até 3 vezes com backoff (300ms → 600ms → 1200ms) em erros transitórios (rede/5xx);
   - não tenta de novo em erros definitivos (409 estoque insuficiente, 404);
   - **circuit breaker**: após 3 falhas consecutivas abre por 15s (fail-fast), evitando sobrecarregar um serviço degradado;
   - ao esgotar as tentativas devolve **503** com mensagem clara — a nota **permanece Aberta** para nova tentativa (recovery comprovado quando o Estoque volta).
6. **Logging** — logger estruturado do chi registra método, rota, IP e duração; erros internos são logados no servidor (nunca expostos ao cliente).
7. **Transações** — `BEGIN/COMMIT/ROLLBACK` garantem atomicidade: a numeração sequencial e a baixa de estoque nunca ficam pela metade.

### Concorrência (opcional a)
A baixa de estoque usa:

```sql
UPDATE products
   SET stock_quantity = stock_quantity - $1
 WHERE code = $2 AND stock_quantity >= $1;
```

O guard `stock_quantity >= $1` torna o UPDATE atômico: se duas notas disputarem o **mesmo** saldo 1, o PostgreSQL serializa as transações e apenas a primeira afeta uma linha — a segunda recebe `ErrInsufficientStock` (409). Comprovado pelo script `scripts/simulate-concurrency.sh` (duas impressões simultâneas: uma 200, outra 409, saldo final 0).

### Idempotência (opcional c)
O `POST /api/invoices/{id}/print` aceita o header `Idempotency-Key`:

- Antes de processar, o sistema consulta a tabela `idempotency_keys` (chave única no banco);
- Se a chave já existe, retorna o **resultado salvo** sem reexecutar a operação (zero efeitos colaterais);
- A chave e o JSON de resposta são gravados somente após o sucesso da operação.

---

## 9. C# / LINQ

**Não se aplica.** A implementação escolhida foi **Golang**, portanto não há uso de LINQ (recurso exclusivo do ecossistema C#/.NET). No lugar de LINQ, o acesso a dados foi feito com **SQL explícito via pgx**.

---

## 10. Funcionalidade opcional de geração de descrição

O sistema inclui um endpoint `/api/products/ai-description` que pode ser usado para sugerir a descrição de um produto via provedor externo. A configuração é feita pelas variáveis de ambiente `AI_BASE_URL`, `AI_API_KEY` e `AI_MODEL`. Caso não haja chave configurada, o serviço retorna um fallback offline com uma descrição genérica, mantendo o fluxo de cadastro funcional.

Caso o provedor esteja indisponível, a chamada cai no fallback automático — o serviço nunca deve quebrar o fluxo de cadastro.

---

## 11. Pontos destacáveis para o vídeo

1. Login JWT → tela de produtos (cadastro de produto) → tela de notas (criação com itens).
2. Impressão com spinner → nota Fechada + saldos atualizados (10 → 8).
3. Bloqueio: nota já Fechada não pode ser impressa.
4. **Falha**: derrubar o Estoque → print retorna erro amigável e nota continua Aberta → subir o Estoque → print funciona (recovery).
5. **Concorrência**: saldo 1, duas notas simultâneas → só uma fecha.
6. **Idempotência**: reenviar o mesmo print com `Idempotency-Key` não baixa estoque duas vezes.
7. Badges de saúde dos serviços na toolbar refletem up/down em tempo real.
8. **Aba Logs**: linha do tempo agregando o que cada microsserviço processou (login, produtos, baixa de estoque, notas, impressões e erros), com atualização automática a cada 3s — cada serviço grava eventos em sua própria tabela `activity_logs` e expõe `GET /api/logs`.
9. **Controle de serviços**: botões na toolbar (ícone `stop`/`play_arrow`) que **param e iniciam** cada microsserviço em tempo real, via supervisor `control` (`backend/control`, porta 8080). Exemplo de apresentação: parar o Estoque pela tela → badge fica vermelho → tentar imprimir nota (erro amigável) → iniciar o Estoque pela tela → nota fecha (recovery). O supervisor é o único dono dos processos: só controla serviços que ele mesmo subiu.

---

## 11.1 Supervisor de serviços (control)

Para permitir iniciar/parar os microsserviços pela interface, um processo Go separado (`backend/control`) roda na porta **8080** e é o **dono dos processos** de Estoque (`:8081`) e Faturamento (`:8082`):

- **Inicia** cada binário com `Setpgid` (grupo de processo próprio), herdando o ambiente e forçando `PORT`, e **aguarda a porta abrir** (health check por TCP, até ~12s) antes de responder sucesso.
- **Para** enviando `SIGTERM` ao grupo de processo (shutdown gracioso dos serviços); se não sair em ~6s, aplica `SIGKILL`.
- Detecta a localização dos binários via `os.Executable()` (irmãos em `backend/{estoque,faturamento}/bin/`), com sobrecarga por variáveis `ESTOQUE_BIN`, `FATURAMENTO_BIN`, `ESTOQUE_PORT`, `FATURAMENTO_PORT`, `CONTROL_PORT`.
- Expõe: `GET /health`, `GET /api/status` e `POST /api/{estoque|faturamento}/{start|stop}` (respostas JSON `{ok, running}`).
- No frontend, o proxy `http-proxy-middleware` encaminha `/api/control/*` → `:8080` (rewrite para `/api/*`). O `ShellComponent` chama `ControlService` e, após a ação, força uma rechecagem imediata do badge de saúde (o polling de 5s continua como rede de segurança). Enquanto o serviço está parado, o polling de health recebe `502` do proxy, que o `HealthService` converte em `down` via `catchError` — comportamento esperado.

`run-all.sh` agora compila os três binários, derruba instâncias antigas e sobe o `control` (que inicia os dois serviços); `stop-all.sh` encerra os três.

---

## 12. Como reproduzir

```bash
bash scripts/setup-db.sh          # 1. banco de dados
bash scripts/run-all.sh --frontend # 2. serviços + frontend (http://localhost:4200)
# login: admin / admin123
```

Testes automatizados:

```bash
bash scripts/e2e-smoke.sh               # 14 asserções
bash scripts/simulate-concurrency.sh    # concorrência
bash scripts/simulate-failure.sh        # falha + recuperação
```