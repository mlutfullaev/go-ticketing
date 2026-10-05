# Ticketing — roadmap

A seat-booking service: venues, events, seat holds, payment, refunds.
You write all the code. Each milestone lists a **spec**, **edge cases**, a **done when** check and hidden **hints**. Open a hint only after you're stuck.

---

## Stack

| Concern | Tool | New vs letterboxd? |
| --- | --- | --- |
| HTTP | stdlib `net/http` (1.22+ patterns) | new — no Echo |
| SQL | `pgx/v5` + `sqlc` | new — no GORM |
| Migrations | `pressly/goose` | new |
| Redis | `redis/go-redis/v9` | new |
| Concurrency helpers | `golang.org/x/sync` (`errgroup`, `singleflight`, `semaphore`) | new |
| Logging | `log/slog` | new |
| Integration tests | `testcontainers-go` | new |
| Config | `caarlos0/env` | known |

## Suggested layout

```
ticketing/
  cmd/api/main.go          wiring + graceful shutdown
  internal/
    catalog/               venues, events, seats, tiers
    booking/               holds (Redis)
    order/                 orders, payment, idempotency
    refund/                policy + refund processing
    worker/                background jobs
    platform/              db, redis, http helpers, middleware
  db/
    migrations/            goose .sql files
    queries/               sqlc .sql files
  sqlc.yaml
  docker-compose.yml
  Makefile                 migrate / generate / run / test
```

## Global conventions

- **Money** is `int64` pence, never `float64`. Currency is GBP only.
- **Time** is stored in UTC (`timestamptz`) and sent as RFC 3339 in JSON.
- **IDs** are UUIDs.
- **Auth** is fake on purpose (you already did JWT): the `X-User-ID` header identifies the user and `X-Role: admin` unlocks admin routes. A middleware puts both into `context`.
- **Errors** always use this JSON shape:
  ```json
  { "error": { "code": "seat_unavailable", "message": "…", "details": { "seat_ids": ["…"] } } }
  ```

---

## Milestone 0 — Skeleton

**Goal:** an empty service that starts, reports its health and shuts down properly.

**Spec**
- `docker-compose.yml` runs Postgres and Redis.
- `GET /healthz` returns `200 {"postgres":"ok","redis":"ok"}`, or `503` with the failing dependency.
- `slog` JSON logger instead of `log`.
- A middleware that logs method, path and duration for every request (no status yet — see M0.5).
- Graceful shutdown on SIGINT/SIGTERM: stop accepting connections, give in-flight requests up to 10s to finish, then close the pools.
- `Makefile` targets: `up`, `migrate`, `generate`, `run`, `test`.

**Done when**
- `docker compose stop redis` makes `/healthz` return 503.
- Add a temporary `GET /slow` that sleeps 5s, call it, press Ctrl+C: the request still completes and the logs show a clean shutdown.

<details><summary>Hints</summary>

1. `signal.NotifyContext` gives you a `ctx` that is cancelled on Ctrl+C. `http.Server.Shutdown(ctx)` is the call that waits for in-flight requests. Read its docs: `ListenAndServe` returns *immediately* with `http.ErrServerClosed`, so `main` must not exit at that point.
2. `/healthz` should use a short `context.WithTimeout` for its pings. Otherwise a dead Redis hangs the health check.
</details>

---

## Milestone 0.5 — Status and request ID in logs (deferred)

Do this later, once M0 feels comfortable.

- **Status in logs.** Problem: `http.ResponseWriter` doesn't remember the status that was written. Fix: pass your own type to `next.ServeHTTP` that embeds `http.ResponseWriter` and overrides `WriteHeader` to record the code.
- **Request ID.** Problem: when two requests log at once, you can't tell which line belongs to which. Fix: read `X-Request-ID` or generate one, store it in `context` (`context.WithValue` + `r.WithContext`), and add it to every log line.

---

## Milestone 1 — Catalogue

**Goal:** admins create venues and events; anyone can browse events and see a seat map.

**Domain rules**
- A **venue** has **sections** (e.g. "Stalls", "Circle"). Each section has `rows × seats_per_row` physical seats.
- An **event** happens at one venue and has **price tiers** (e.g. "Premium" £80, "Standard" £45). Every section is assigned to exactly one tier.
- A seat is *physical* (it belongs to the venue). Its *status and price* belong to one event. The same seat can be sold for Monday's show and free for Tuesday's.
- Event status: `draft → on_sale → finished`, or `cancelled` from `draft` or `on_sale`.
- An event has a sales window (`sales_open_at`, `sales_close_at`). `sales_close_at` must be ≤ `starts_at`.

**Entities** (you design the tables and constraints)
- `venues`, `sections`, `seats` — a seat is identified by (venue, section, row, number), and that combination must be unique.
- `events`, `price_tiers`, `event_seats` — `event_seats` holds the per-event status (`available` / `sold`) and tier.

**API**

| Method | Path | Who | Notes |
| --- | --- | --- | --- |
| POST | `/venues` | admin | body includes sections with `rows`, `seats_per_row`; generates the seats |
| POST | `/events` | admin | creates a `draft` with tiers → sections mapping |
| POST | `/events/{id}/publish` | admin | `draft → on_sale`; fails unless every section has a tier |
| GET | `/events` | anyone | filter `?city=&from=&to=`, only `on_sale`, paginated |
| GET | `/events/{id}` | anyone | event + tiers with prices |
| GET | `/events/{id}/seats` | anyone | `[{seat_id, section, row, number, tier, price_pence, status}]` |

**Edge cases**
- Creating a venue with 2,000 seats must not take 2,000 separate `INSERT` round trips.
- Publishing an event creates its `event_seats` rows. If that fails halfway, nothing should be left behind.
- `from > to`, a missing city, and past events in the list.

**Done when**
- You can create a venue with 3 sections, create and publish an event, and see the full seat map.
- All SQL is in `db/queries/*.sql`, with code generated by sqlc. No hand-written `rows.Scan` in services.

<details><summary>Hints</summary>

1. For bulk inserts, look at sqlc's `:copyfrom` (Postgres `COPY`), or at a single `INSERT … SELECT FROM generate_series(…)`.
2. "Nothing left behind" means a transaction. With pgx and sqlc, `pool.Begin(ctx)` gives you a `pgx.Tx`, and `queries.WithTx(tx)` gives you queries bound to it. Think about *which layer* opens the transaction. The repo doesn't know the use case, and the handler shouldn't know about SQL.
3. Pagination: offset is easy but inconsistent when rows change underneath you. Keyset (cursor) pagination on `(starts_at, id)` is the grown-up version. Do offset first, then switch.
</details>

---

## Milestone 2 — Holds (Redis) ⚠️ the hardest one

**Goal:** a user reserves seats for 10 minutes. Nobody else can take them in that time.

**Domain rules**
- A hold covers 1–6 seats of **one** event, and it is **all or nothing**.
- A user may have at most 6 held seats per event across all their holds.
- Holds are only allowed while the event is `on_sale` and inside its sales window.
- A seat can be held only if it is `available` in Postgres **and** not held in Redis.
- Hold TTL is 10 minutes. The expiry is automatic, with no cleanup code.
- **Postgres knows `available`/`sold`. Redis knows `held`.** The seat map from M1 now merges both.

**API**

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/events/{id}/holds` | `{seat_ids:[…]}` → `201 {hold_id, seat_ids, expires_at}` |
| GET | `/holds/{id}` | only the owner; `404` once expired |
| DELETE | `/holds/{id}` | releases the seats immediately |

A conflict returns `409` with `details.seat_ids` listing *which* seats were taken, so the frontend can highlight them.

**Edge cases**
- Two users grab overlapping seat sets at the same moment. Exactly one gets all of them; the other gets nothing.
- The same seat appears twice in `seat_ids`.
- The user already holds 4 seats and asks for 3 more.
- A seat id from a *different* event.
- Redis is down. What should the API return? Don't let it hang.

**Done when**
- A test starts **100 goroutines**, all holding the same seat. Exactly 1 succeeds, and `go test -race` is clean.
- A test with overlapping sets (A wants seats 1,2,3 and B wants 3,4,5) never ends with both holding seat 3.

<details><summary>Hint 1 — the problem</summary>

`SET key val NX EX 600` is atomic for **one** key. Looping over 4 seats with 4 `SET NX` calls is not atomic: you can win seats 1–2, lose seat 3, and you're now holding a partial set. Undoing it afterwards leaves a window where others see a half-hold. Find something in Redis that runs several commands as one step.
</details>

<details><summary>Hint 2 — the tool</summary>

Redis runs **Lua scripts** atomically, with nothing interleaved. In go-redis: `redis.NewScript(src)` and `script.Run(ctx, rdb, keys, args...)`. The script checks every key first, then sets them all or none. Note that `MULTI/EXEC` is *not* enough, because it can't branch on a value it read.
</details>

<details><summary>Hint 3 — key design</summary>

Think about which lookups you need:
- "who holds seat X of event E?" → one key per (event, seat)
- "what's in hold H, and whose is it?" → one key per hold
- "how many seats does user U hold for event E?" → one more key

All of them must share the same TTL, or they drift apart.
</details>

<details><summary>Hint 4 — Go side of the test</summary>

`sync.WaitGroup` to wait, `atomic.Int32` to count successes, and a `chan struct{}` closed once so that all goroutines start at the same instant. Without the start gate, most goroutines finish before the last ones start, and the test proves little.
</details>

---

## Milestone 3 — Orders & payment

**Goal:** turn a hold into a paid order with tickets, safely, even when things fail halfway.

**Domain rules**
- `POST /orders {hold_id}` creates a `pending` order. It **snapshots prices** (if the admin changes a tier price later, this order doesn't change) and shares the hold's `expires_at`.
- Paying runs these steps: charge the card → in one DB transaction, lock the seats, check they're still `available`, mark them `sold`, mark the order `paid`, create tickets → delete the Redis hold.
- A ticket has a unique, unguessable `code` (think QR code).
- Order status: `pending → paid → refunded`, or `pending → expired`, or `pending → failed`.

**Payment provider** (fake, behind an interface, so tests can control it)

```go
type PaymentProvider interface {
	Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
	Refund(ctx context.Context, chargeID string, amountPence int64) error
}
```

The fake is configurable: latency, decline rate, and "hang until the context is cancelled". Call it with a 5s `context.WithTimeout`.

**Idempotency**
- `POST /orders/{id}/pay` requires an `Idempotency-Key` header.
- Same key + same body → return the **stored** response and don't charge again.
- Same key + a different body → `422`.
- Same key while the first request is still running → `409`, or wait (your choice; justify it).

**API**

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/orders` | from a hold |
| POST | `/orders/{id}/pay` | idempotent |
| GET | `/orders/{id}` | owner only; includes tickets once paid |
| GET | `/me/orders` | paginated |

**Edge cases** (this is where the real learning is)
- The hold expires *while* the charge is in flight.
- The charge succeeds, then the DB transaction fails. The customer has paid for nothing, so you must **refund** (a compensating action). Log it loudly.
- The provider times out. Did the charge happen or not? You don't know.
- The client retries after a network error on the response.

**Done when**
- Paying twice with the same key charges exactly once (assert on the fake's call count).
- A test where the fake charges successfully and the DB step is forced to fail results in exactly one `Refund` call.
- Domain errors are types, not strings: e.g. `*SeatsUnavailableError{SeatIDs}` matched with `errors.As` in the handler, and sentinel errors matched with `errors.Is`.

<details><summary>Hints</summary>

1. `SELECT … FROM event_seats WHERE … FOR UPDATE` locks those rows until commit. A second payer for the same seat waits, then sees `sold`.
2. Charge-then-lock vs lock-then-charge: holding DB row locks during a slow external HTTP call is a classic mistake. Charge first, and compensate on failure.
3. The idempotency record needs a status (`in_progress` / `done`), a hash of the request body and the stored response. Where does it live, Postgres or Redis? Both work. Pick one, and write down what you lose.
4. If the pay handler returns early because the *client* disconnected, `r.Context()` is cancelled. Should that cancel an in-flight charge? Look at `context.WithoutCancel`.
</details>

---

## Milestone 4 — Background worker

**Goal:** things that happen without a request: expiring stale orders, and fixing payments left in an unknown state.

**Spec**
- Every 30s, mark `pending` orders past `expires_at` as `expired`.
- Every 60s, find orders stuck in `paying` for more than 2 minutes and ask the provider what happened. Settle them as paid, or refund. (Add a `GetCharge` method to the interface.)
- Run the workers and the HTTP server under one `errgroup`, sharing the shutdown `ctx`. If one crashes, everything shuts down cleanly.
- It must be safe to run **two instances** of the service at once.

**Done when**
- Two instances run against the same DB, and each expired order is processed exactly once (log it and count it).
- Tests don't `time.Sleep` for 30s. Time is injected.

<details><summary>Hints</summary>

1. `FOR UPDATE SKIP LOCKED` with `LIMIT 100` lets many workers split a queue table without stepping on each other. It's the most useful Postgres trick for background jobs.
2. Inject a clock: `type Clock interface{ Now() time.Time }`. Production uses `time.Now`; tests use a fake you move forward by hand. Also look at `testing/synctest` in the stdlib for testing tickers.
3. Inside the worker loop, `select` on both `ticker.C` and `ctx.Done()`. Forgetting the second one is why workers won't stop.
</details>

---

## Milestone 5 — Refunds & cancellation

**Goal:** real business rules, and bulk processing.

**Refund policy** (a pure function, no DB and no time.Now inside)

| Time before event start | Refund |
| --- | --- |
| more than 7 days | 100% |
| 48h to 7 days | 50% |
| less than 48h | 0% — not cancellable |
| event cancelled by organiser | always 100% |

Round **down** to whole pence.

**Spec**
- `POST /orders/{id}/cancel` (by the user) applies the policy, refunds, sets the seats back to `available` and voids the tickets.
- `POST /events/{id}/cancel` (admin) cancels the event and refunds **every** paid order at 100%:
  - uses a pool of at most 5 concurrent refunds (the provider rate-limits you)
  - retries a failed refund with exponential backoff, up to 5 attempts
  - is **resumable**: if the process dies halfway, restarting continues where it stopped, and no one is refunded twice
- `GET /events/{id}/cancellation` shows progress: `{total, refunded, failed, pending}`.

**Done when**
- Table-driven tests cover every boundary: exactly 7d, 7d − 1s, exactly 48h, 48h − 1s, an odd pence amount.
- Killing the process mid-cancellation and restarting it ends with every order refunded exactly once.

<details><summary>Hints</summary>

1. The policy's signature decides whether it's testable: `func RefundAmount(paidPence int64, eventStart, now time.Time, reason Reason) (int64, error)`.
2. For bounded concurrency, `errgroup.Group.SetLimit(5)` is the simplest. A buffered `chan struct{}` used as a semaphore is the manual version; write it once to understand it.
3. Resumable means the *state lives in the DB*, not in goroutines. Each order's refund status is a row, and the bulk job is only a loop over the rows not yet done. This is the same `SKIP LOCKED` pattern as M4.
</details>

---

## Milestone 6 — Hardening

**Spec**
- **Rate limiting** in Redis as middleware: `POST /events/{id}/holds` allows 10 per minute per user. Return `429` with a `Retry-After` header. Build a fixed window (`INCR` + `EXPIRE`) first, then a sliding window (sorted set).
- **Caching** `GET /events/{id}` in Redis, invalidated on publish, cancel and price change. Protect it from a stampede, where 1,000 requests miss the cache at the same moment.
- **Integration tests** with testcontainers: real Postgres and Redis, migrations applied, and the M2/M3 concurrency tests run against them.
- **Profiling**: expose `net/http/pprof` on a separate admin port, run a load test and look at a CPU profile.

**Done when**
- `go test -race ./...` runs green against real containers.
- A load test (`vegeta` or `hey`, both written in Go) of 500 concurrent hold attempts on 100 seats ends with no double-held seat and no 5xx.

<details><summary>Hints</summary>

1. Stampede protection: `golang.org/x/sync/singleflight` merges concurrent identical calls into one.
2. For testcontainers, use `TestMain` to start the containers once per package, not once per test.
3. pprof: `go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10`, then `top` and `web`.
</details>

---

## Stretch goals

- **Live seat map**: a WebSocket endpoint pushes seat changes as they happen. Redis pub/sub fans them out across instances. (`coder/websocket`)
- **Transactional outbox**: when an order is paid, write an `order_paid` event in the *same* transaction, and have a worker publish it to a Redis Stream (e.g. for a fake email sender).
- **Waiting queue**: when an event opens, put users in a queue and let them in N per second.
- **OpenAPI spec**, and generating the handlers from it with `oapi-codegen`.
