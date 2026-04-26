# Migrate local Postgres → NeonDB

## Context

The `investment-intelligence` project (Manning book ch. 3) currently runs a local
Postgres 16 container via `src/docker-compose.yml`. The Go backend connects
through `DATABASE_URL` and auto-applies schema via golang-migrate on startup. The
user wants to replace the local Postgres container with a managed NeonDB
(serverless Postgres) instance.

**Decisions locked in during review:**

- Provision Neon via the Neon CLI directly (project is not linked to Vercel and
  there is no plan to link it as part of this task).
- Clean-slate migration — no `pg_dump`/restore. golang-migrate will create the
  empty schema on first backend startup against Neon.
- The local `postgres` service in Docker Compose is removed entirely. Neon
  becomes the only database, for local dev and beyond.

**Why this is low-risk:**

- Schema is trivially portable: 3 tables, `uuid-ossp` extension, no JSON/vector/custom types
  (`src/backend/migrations/000001_init_schema.up.sql:1-30`).
- Query layer is raw SQL via `database/sql` + `lib/pq` — no ORM, no driver lock-in.
- Only one env var drives the connection: `DATABASE_URL`
  (`src/backend/cmd/server/main.go:24-27`).
- Neon supports `uuid-ossp`, `sslmode=require`, and standard `postgres://` URIs —
  `lib/pq` works unchanged.

**Intended outcome:** `docker compose up` in `src/` starts three services
(data-service, backend, frontend) that all connect — directly or via the backend
— to a Neon Postgres instance. A developer cloning the repo only needs to copy
`.env.example` → `.env` and paste in a Neon connection string.

## Current state (verified)

| Concern            | File                                               | Current behavior                                                             |
| ------------------ | -------------------------------------------------- | ---------------------------------------------------------------------------- |
| Local DB container | `src/docker-compose.yml:2-16`                      | Postgres 16, volume `pgdata`, credentials in file                            |
| Backend DB URL     | `src/docker-compose.yml:35`                        | `postgres://portfolio:portfolio_dev@postgres:5432/portfolio?sslmode=disable` |
| Connection open    | `src/backend/cmd/server/main.go:33-42`             | `sql.Open("postgres", dbURL)` + `Ping`                                       |
| Migration runner   | `src/backend/cmd/server/main.go:104-119`           | golang-migrate, reads `file://migrations` on startup                         |
| Schema             | `src/backend/migrations/000001_init_schema.up.sql` | `uuid-ossp`, `stocks`, `transactions`, `prices_cache`                        |

## Step-by-step plan

### Step 1 — Provision Neon (user-driven, interactive)

Run from the user's shell (not automated — these require an interactive auth
flow and one-time choices):

```bash
# Install once
npm i -g neonctl   # or: brew install neonctl

# Auth and create the project
neonctl auth
neonctl projects create --name investment-intelligence
```

Capture the project id from the output, then:

```bash
# Create the database + role (names match the existing docker-compose defaults)
neonctl databases create --name portfolio --project-id <id>
neonctl roles create --name portfolio --project-id <id>

# Print the connection string (will include ?sslmode=require)
neonctl connection-string --project-id <id> \
  --database-name portfolio --role-name portfolio
```

The resulting URL looks like:
`postgres://portfolio:<pw>@ep-xxx.eu-central-1.aws.neon.tech/portfolio?sslmode=require`

**Note:** Neon's default database is `neondb` and default role is `neondb_owner`.
The commands above explicitly create `portfolio`/`portfolio` to match the
existing project conventions — but if creating extra roles/DBs is friction, using
the defaults is equally fine; only `DATABASE_URL` has to change.

### Step 2 — Create `src/.env.example` and `src/.env`

**New file — `src/.env.example` (committed):**

```
# Neon Postgres connection string. Create one via `neonctl connection-string ...`
DATABASE_URL=postgres://USER:PASSWORD@HOST/DATABASE?sslmode=require
```

**New file — `src/.env` (gitignored, developer-local):**
Paste the real connection string from Step 1.

Verify `.env` is git-ignored. `git check-ignore src/.env` should print the path.
If not, add `src/.env` to the repo's `.gitignore`.

### Step 3 — Rewrite `src/docker-compose.yml`

Delete the `postgres` service, delete the `pgdata` volume, and change the
`backend` service to read `DATABASE_URL` from the host environment. Docker
Compose auto-loads a sibling `.env` file so `${DATABASE_URL}` resolves from
`src/.env`.

Exact edits:

- **Remove lines 2-16** (the whole `postgres:` service block).
- **Remove lines 53-54** (the `volumes:` / `pgdata:` definition).
- **Replace the `backend` service (lines 30-41)** with:

```yaml
backend:
  build: ./backend
  ports:
    - "8080:8080"
  environment:
    DATABASE_URL: ${DATABASE_URL:?DATABASE_URL is required - set it in src/.env}
    DATA_SERVICE_URL: http://data-service:8000
  depends_on:
    data-service:
      condition: service_healthy
```

Key points:

- `${DATABASE_URL:?...}` makes Compose fail loudly if the env var is missing.
- `depends_on` no longer references `postgres` — only `data-service` remains.
- No change to `data-service` or `frontend` blocks.

### Step 4 — Start the stack and let golang-migrate create the schema

Nothing to script here — this is the first verification:

```bash
cd src
docker compose up --build backend
```

Expected log lines from the backend container:

- `Connected to database`
- `Migrations complete`

The existing runner in `src/backend/cmd/server/main.go:104-119` reads the
migrations baked into the container image and applies them to whatever URL
`DATABASE_URL` points at. No code change needed.

### Step 5 — Document the new flow in `src/CLAUDE.md`

Update the "Full stack (Docker)" section (`src/CLAUDE.md:30-37`) to note the
new prerequisite:

- Copy `.env.example` → `.env` and fill in `DATABASE_URL` (Neon connection string).
- The local `postgres` container is gone; Neon is the only database.
- `docker compose down -v` no longer wipes a local volume — schema lives on Neon
  and persists across runs. To reset, use a fresh Neon branch or drop tables
  with `psql "$DATABASE_URL"`.

No change required to the "Database Schema" or "Environment Variables" sections
beyond noting that `DATABASE_URL` is sourced from `.env`.

## Files to modify

| File                       | Type             | Change                                                                                                                                              |
| -------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `src/docker-compose.yml`   | Edit             | Remove `postgres` service + `pgdata` volume; change `backend.environment.DATABASE_URL` to `${DATABASE_URL:?...}`; drop `postgres` from `depends_on` |
| `src/.env.example`         | New              | Commit a template with `DATABASE_URL=postgres://USER:PASSWORD@HOST/DATABASE?sslmode=require`                                                        |
| `src/.env`                 | New (gitignored) | Developer-local; paste Neon URL here                                                                                                                |
| `src/.gitignore` (or root) | Verify/edit      | Confirm `src/.env` is ignored                                                                                                                       |
| `src/CLAUDE.md`            | Edit             | Short update to the "Full stack (Docker)" section describing the Neon prereq                                                                        |

**No code changes** in `backend/`, `data-service/`, or `frontend/`. The Go code
is driver-agnostic about whether it talks to local Postgres or Neon.

## Verification

1. **Schema was created on Neon:**

   ```bash
   psql "$DATABASE_URL" -c '\dt'
   # expect: stocks, transactions, prices_cache, schema_migrations
   ```

2. **Backend starts cleanly against Neon** (see Step 4 log lines).

3. **End-to-end smoke test with all services:**

   ```bash
   cd src && docker compose up --build
   # in another shell:
   curl -s http://localhost:8080/health
   curl -s -X POST http://localhost:8080/api/transactions \
     -H 'Content-Type: application/json' \
     -d '{"ticker":"AAPL","transaction_type":"buy","shares":10,"price_per_share":150,"transaction_date":"2026-04-01"}'
   curl -s http://localhost:8080/api/portfolio
   # open http://localhost:3000 in a browser to verify the Next.js app renders the transaction
   ```

4. **Row persists across container restarts:**

   ```bash
   docker compose down
   docker compose up -d
   curl -s http://localhost:8080/api/transactions   # should still contain the AAPL row
   ```

5. **Compose fails loudly when `DATABASE_URL` is missing** (belt-and-suspenders
   check on the `${VAR:?...}` guard):
   ```bash
   unset DATABASE_URL && docker compose config
   # expect: error mentioning "DATABASE_URL is required"
   ```

## Rollback plan

Git-revert the `docker-compose.yml`, `.env.example`, and `CLAUDE.md` changes.
The Neon project can be left running (free tier) or deleted with
`neonctl projects delete <id>`. Since this is a clean-slate migration, nothing
local is destroyed during the migration itself — the existing `pgdata` volume
stays intact until a `docker compose down -v` is run against the _current_
compose file.
