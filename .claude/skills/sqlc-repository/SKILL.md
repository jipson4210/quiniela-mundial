---
name: sqlc-repository
description: How to implement repositories using sqlc for type-safe SQL in Go. Use this skill when creating any new repository, when writing SQL queries that need to be compiled into Go code, when adding migrations, or when working on anything under /internal/infrastructure/persistence/postgres/. Apply this whenever the task involves database access in the Go backend — sqlc is the chosen tool and must be used instead of GORM or raw sqlx.
---

# sqlc Repository Pattern

Repositorios con `sqlc` (no GORM). Generamos Go tipado desde SQL plano.

## Por qué sqlc y no GORM

- **Clean Architecture friendly:** `sqlc` no contamina entidades del dominio con tags ni métodos
- **Type-safe:** errores de tipo en SQL → errores en compilación, no en runtime
- **SQL es la fuente de verdad:** mantienes control total de las queries
- **Sin reflection en hot path**

## Instalación

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Estructura

```
/
├── migrations/            # golang-migrate
│   ├── 0001_users.up.sql
│   ├── 0001_users.down.sql
│   └── ...
├── queries/               # archivos .sql con anotaciones de sqlc
│   ├── users.sql
│   ├── pools.sql
│   └── ...
├── sqlc.yaml              # configuración
└── internal/infrastructure/persistence/postgres/
    ├── sqlc/              # GENERADO por sqlc — no editar manualmente
    │   ├── db.go
    │   ├── models.go
    │   └── *.sql.go
    ├── user_repo.go       # Repositorios que envuelven sqlc.Queries
    ├── pool_repo.go
    └── ...
```

## `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/infrastructure/persistence/postgres/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_db_tags: false
        emit_interface: true
        emit_prepared_queries: false
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "uuid"
            go_type: "string"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

## Migración: ejemplo

```sql
-- migrations/0001_users.up.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(50) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    verified_at   TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
```

```sql
-- migrations/0001_users.down.sql
DROP TABLE IF EXISTS users;
```

## Query: ejemplo

```sql
-- queries/users.sql

-- name: CreateUser :one
INSERT INTO users (email, password_hash, display_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsersByPool :many
SELECT u.*
FROM users u
JOIN pool_members pm ON pm.user_id = u.id
WHERE pm.pool_id = $1
ORDER BY pm.joined_at ASC;
```

**Anotaciones clave:**
- `:one` — devuelve una sola fila (error si no hay)
- `:many` — devuelve un slice
- `:exec` — no devuelve filas (INSERT/UPDATE/DELETE)
- `:execrows` — devuelve número de filas afectadas
- `:batchmany` / `:batchone` — operaciones batch con pgx

## Generación

```bash
make sqlc
# o:
sqlc generate
```

Esto genera `internal/infrastructure/persistence/postgres/sqlc/*.go` con structs y funciones tipadas:

```go
// GENERADO por sqlc — no editar
type User struct {
    ID           string
    Email        string
    PasswordHash string
    DisplayName  string
    CreatedAt    time.Time
    VerifiedAt   *time.Time
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
    // ...
}
```

## Repositorio: envoltura del dominio

El `Queries` de sqlc no es el repositorio del dominio — es su backend. Envuélvelo:

```go
// internal/infrastructure/persistence/postgres/user_repo.go
package postgres

import (
    "context"
    "errors"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/<repo>/internal/domain/user"
    "github.com/<repo>/internal/infrastructure/persistence/postgres/sqlc"
)

type UserRepo struct {
    q *sqlc.Queries
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
    return &UserRepo{q: sqlc.New(db)}
}

// Save implementa user.Repository.Save
func (r *UserRepo) Save(ctx context.Context, u *user.User) error {
    _, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
        Email:        u.Email().String(),
        PasswordHash: u.PasswordHash(),
        DisplayName:  u.DisplayName(),
    })
    return err
}

func (r *UserRepo) FindByEmail(ctx context.Context, email shared.Email) (*user.User, error) {
    row, err := r.q.GetUserByEmail(ctx, email.String())
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, user.ErrNotFound
        }
        return nil, err
    }
    return toDomain(row), nil
}

// toDomain mapea el row de sqlc a la entidad del dominio
func toDomain(row sqlc.User) *user.User {
    email, _ := shared.NewEmail(row.Email)
    return user.Reconstruct(
        user.ID(row.ID),
        email,
        row.PasswordHash,
        row.DisplayName,
        row.CreatedAt,
        row.VerifiedAt,
    )
}
```

> **Nota sobre `Reconstruct`:** las entidades del dominio tienen dos formas de instanciarse:
> - `NewUser(...)` — constructor que valida invariantes (uso desde casos de uso)
> - `Reconstruct(...)` — rehidrata una entidad desde persistencia, **sin re-validar** (uso solo desde repos)

```go
// internal/domain/user/user.go
func Reconstruct(
    id ID,
    email shared.Email,
    passwordHash, displayName string,
    createdAt time.Time,
    verifiedAt *time.Time,
) *User {
    return &User{
        id:           id,
        email:        email,
        passwordHash: passwordHash,
        displayName:  displayName,
        createdAt:    createdAt,
        verifiedAt:   verifiedAt,
    }
}
```

## Transacciones

`pgx` con `sqlc` soporta transacciones vía `pgx.Tx`:

```go
// Repositorio con soporte de tx
type UserRepo struct {
    db *pgxpool.Pool
}

func (r *UserRepo) WithTx(tx pgx.Tx) user.Repository {
    return &UserRepo{q: sqlc.New(tx)}
}

// Caso de uso con tx
func (uc *AcceptInvitation) Execute(ctx context.Context, in Input) error {
    tx, err := uc.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    users := uc.users.WithTx(tx)
    pools := uc.pools.WithTx(tx)

    // ... operaciones
    if err := users.Save(ctx, newUser); err != nil {
        return err
    }
    if err := pools.AddMember(ctx, poolID, newUser.ID()); err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

## Arrays en PostgreSQL (para BracketPrediction)

```sql
-- queries/bracket_predictions.sql
-- name: UpsertBracketPrediction :one
INSERT INTO bracket_predictions (
    user_id, pool_id, tournament_id,
    teams_to_round_of_32, teams_to_round_of_16, teams_to_quarter_final,
    teams_to_semi_final, teams_to_final, third_place_winner, champion
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (user_id, pool_id, tournament_id) DO UPDATE SET
    teams_to_round_of_32 = EXCLUDED.teams_to_round_of_32,
    teams_to_round_of_16 = EXCLUDED.teams_to_round_of_16,
    teams_to_quarter_final = EXCLUDED.teams_to_quarter_final,
    teams_to_semi_final = EXCLUDED.teams_to_semi_final,
    teams_to_final = EXCLUDED.teams_to_final,
    third_place_winner = EXCLUDED.third_place_winner,
    champion = EXCLUDED.champion,
    updated_at = now()
RETURNING *;
```

`sqlc` genera los params con `[]string` para los arrays `UUID[]`. Mapeo:

```go
func toTeamIDs(strs []string) []team.ID {
    ids := make([]team.ID, len(strs))
    for i, s := range strs {
        ids[i] = team.ID(s)
    }
    return ids
}

func fromTeamIDs(ids []team.ID) []string {
    strs := make([]string, len(ids))
    for i, id := range ids {
        strs[i] = string(id)
    }
    return strs
}
```

## Tests con testcontainers

```go
// internal/infrastructure/persistence/postgres/user_repo_test.go
func TestUserRepo_FindByEmail(t *testing.T) {
    ctx := context.Background()
    pgC, err := postgrescontainer.Run(ctx, "postgres:16-alpine",
        postgrescontainer.WithDatabase("test"),
        postgrescontainer.WithUsername("test"),
        postgrescontainer.WithPassword("test"),
        testcontainers.WithWaitStrategy(wait.ForLog("ready to accept connections").WithOccurrence(2)),
    )
    require.NoError(t, err)
    defer pgC.Terminate(ctx)

    dsn, _ := pgC.ConnectionString(ctx, "sslmode=disable")
    db, err := pgxpool.New(ctx, dsn)
    require.NoError(t, err)

    runMigrations(t, dsn)

    repo := NewUserRepo(db)

    email, _ := shared.NewEmail("test@example.com")
    u, _ := user.NewUser(email, "hash", "Test User", time.Now())
    err = repo.Save(ctx, u)
    require.NoError(t, err)

    found, err := repo.FindByEmail(ctx, email)
    require.NoError(t, err)
    assert.Equal(t, "Test User", found.DisplayName())
}
```

## Makefile útil

```makefile
.PHONY: sqlc migrate-up migrate-down test-db

sqlc:
	sqlc generate

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1

new-migration:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

test-db:
	docker run --rm -d --name test-pg -p 5433:5432 \
		-e POSTGRES_PASSWORD=test postgres:16-alpine
```

## Antipatrones

❌ **Editar archivos en `sqlc/` generados.** Se sobreescriben. Cambios van en `queries/*.sql`.

❌ **Exportar tipos de `sqlc` fuera del paquete `postgres`.** Son detalle de implementación.

❌ **Hacer JOIN complejos en `:many` cuando un caso de uso ya sabe qué entidades cargar.** Mejor cargar por separado y componer en aplicación.

❌ **Usar transacciones que cruzan múltiples HTTP requests.** Las transacciones son por caso de uso.
