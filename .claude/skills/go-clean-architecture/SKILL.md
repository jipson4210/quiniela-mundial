---
name: go-clean-architecture
description: How to structure Go code following Clean Architecture for the Quiniela Mundial project. Use this skill whenever creating new Go packages, modules, or features in this repo — whether for new domain entities, use cases, repositories, HTTP handlers, or any backend code. Apply this for ALL backend Go work, even if the user does not explicitly mention "clean architecture", because the entire codebase depends on these layering rules being respected.
---

# Go Clean Architecture — Quiniela Mundial

Reglas de oro al escribir código Go en este repositorio. Si vas a crear cualquier archivo `.go`, lee esto primero.

## La regla fundamental

**Las dependencias siempre apuntan hacia adentro.**

```
interfaces ──→ application ──→ domain ←── infrastructure
```

`domain` no importa **nada** de las otras capas. Si te ves importando `github.com/<repo>/internal/infrastructure/...` dentro de `internal/domain/...`, **estás mal**.

## Estructura del repo

```
internal/
├── domain/             # Capa interna. Solo stdlib + paquetes utilitarios sin lógica de negocio.
├── application/        # Casos de uso. Importa domain. NO importa infrastructure ni interfaces.
├── infrastructure/     # Adapters. Importa domain (para implementar ports).
└── interfaces/         # HTTP, CLI, jobs. Importa application.
```

## Convenciones de paquetes

### Domain packages

Un paquete por agregado. Cada paquete tiene:
- Una entidad (struct con métodos)
- Sus value objects
- Su interface `Repository`
- Sus errores como variables (`var ErrXxx = errors.New(...)`)

Ejemplo: `internal/domain/prediction/`
```
prediction/
├── match_prediction.go     # Entity
├── bracket_prediction.go   # Entity
├── repository.go           # Interfaces
├── errors.go               # var Err...
└── *_test.go               # Tests del dominio
```

### Application packages

Separar comandos de queries (CQRS ligero):

```
application/
├── commands/
│   ├── submit_match_prediction.go
│   ├── create_pool.go
│   └── ...
└── queries/
    ├── get_pool_ranking.go
    ├── list_user_predictions.go
    └── ...
```

Cada caso de uso es **una struct** con un único método `Execute`:

```go
type SubmitMatchPrediction struct {
    predictions prediction.Repository
    matches     match.Repository
    clock       Clock
}

func NewSubmitMatchPrediction(
    predictions prediction.Repository,
    matches match.Repository,
    clock Clock,
) *SubmitMatchPrediction {
    return &SubmitMatchPrediction{predictions, matches, clock}
}

type SubmitMatchPredictionInput struct {
    UserID    string
    PoolID    string
    MatchID   string
    HomeGoals int
    AwayGoals int
}

func (uc *SubmitMatchPrediction) Execute(
    ctx context.Context,
    in SubmitMatchPredictionInput,
) error {
    // Orquesta entidades y repos del dominio
}
```

## Reglas estrictas

### 1. Sin tags de ORM en domain

❌ Mal:
```go
package prediction

type MatchPrediction struct {
    ID        string `gorm:"primaryKey" db:"id"`
    UserID    string `gorm:"index" db:"user_id"`
}
```

✅ Bien:
```go
package prediction

type MatchPrediction struct {
    id        ID
    userID    user.ID
    // campos privados, expuestos por getters
}

func (p *MatchPrediction) ID() ID { return p.id }
func (p *MatchPrediction) UserID() user.ID { return p.userID }
```

Los tags de DB van en los structs generados por `sqlc` en `infrastructure/persistence/postgres/sqlc/`.

### 2. Constructores que validan

Toda entidad se crea con un constructor que valida invariantes:

```go
func NewMatchPrediction(
    userID user.ID,
    poolID pool.ID,
    matchID match.ID,
    homeGoals, awayGoals int,
    now time.Time,
    match *match.Match,
) (*MatchPrediction, error) {
    if now.After(match.KickoffAt()) || now.Equal(match.KickoffAt()) {
        return nil, ErrPredictionWindowClosed
    }
    if homeGoals < 0 || awayGoals < 0 {
        return nil, ErrInvalidScore
    }
    if homeGoals > 30 || awayGoals > 30 {
        return nil, ErrUnreasonableScore
    }
    return &MatchPrediction{
        id:        newID(),
        userID:    userID,
        poolID:    poolID,
        matchID:   matchID,
        homeGoals: homeGoals,
        awayGoals: awayGoals,
        submittedAt: now,
        updatedAt:   now,
    }, nil
}
```

### 3. Errores tipados

```go
// internal/domain/prediction/errors.go
package prediction

import "errors"

var (
    ErrPredictionWindowClosed = errors.New("prediction window is closed")
    ErrInvalidScore           = errors.New("invalid score")
    ErrUnreasonableScore      = errors.New("score too high")
    ErrNotFound               = errors.New("prediction not found")
)
```

En `interfaces/http/`, mapear con `errors.Is`:
```go
err := uc.Execute(ctx, in)
switch {
case errors.Is(err, prediction.ErrPredictionWindowClosed):
    c.JSON(409, gin.H{"error": "prediction window closed"})
case errors.Is(err, prediction.ErrInvalidScore):
    c.JSON(400, gin.H{"error": "invalid score"})
case err != nil:
    c.JSON(500, gin.H{"error": "internal error"})
}
```

### 4. Context siempre

Cualquier función que puede bloquear o cruzar capa lleva `context.Context` como primer parámetro:

```go
func (r *MatchRepo) FindByID(ctx context.Context, id match.ID) (*match.Match, error)
func (uc *SubmitMatchPrediction) Execute(ctx context.Context, in Input) error
```

### 5. Ports en domain, adapters en infrastructure

Ejemplo: el dominio declara qué necesita.

```go
// internal/domain/match/repository.go
package match

type Repository interface {
    FindByID(ctx context.Context, id ID) (*Match, error)
    Save(ctx context.Context, m *Match) error
}
```

La infraestructura implementa.

```go
// internal/infrastructure/persistence/postgres/match_repo.go
package postgres

type MatchRepo struct {
    q *sqlc.Queries
}

func NewMatchRepo(db *pgxpool.Pool) *MatchRepo {
    return &MatchRepo{q: sqlc.New(db)}
}

func (r *MatchRepo) FindByID(ctx context.Context, id match.ID) (*match.Match, error) {
    row, err := r.q.GetMatch(ctx, string(id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, match.ErrNotFound
        }
        return nil, err
    }
    return toDomain(row), nil
}

func toDomain(row sqlc.Match) *match.Match {
    // Mapear de struct de sqlc a entidad de dominio
}
```

### 6. Inyección de dependencias con Wire

```go
// cmd/api/wire.go
//go:build wireinject

package main

import (
    "github.com/google/wire"
    // ...
)

func InitializeApp(cfg Config) (*App, error) {
    wire.Build(
        // Infrastructure
        newPgxPool,
        postgres.NewMatchRepo,
        wire.Bind(new(match.Repository), new(*postgres.MatchRepo)),
        // ... etc

        // Application
        commands.NewSubmitMatchPrediction,

        // Interfaces
        handlers.NewPredictionsHandler,
        newRouter,
        newApp,
    )
    return nil, nil
}
```

Esto se genera con `wire ./cmd/api`. Compile-time, sin reflection.

## Functional Options para configuración

Para constructores con muchos parámetros opcionales:

```go
type Server struct {
    port    int
    timeout time.Duration
    logger  Logger
}

type Option func(*Server)

func WithPort(p int) Option           { return func(s *Server) { s.port = p } }
func WithTimeout(t time.Duration) Option { return func(s *Server) { s.timeout = t } }
func WithLogger(l Logger) Option      { return func(s *Server) { s.logger = l } }

func NewServer(opts ...Option) *Server {
    s := &Server{port: 8080, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

## Checklist al crear código nuevo

Antes de hacer commit, verificar:

- [ ] El paquete está en la capa correcta
- [ ] No hay imports "hacia afuera" desde domain
- [ ] Entidades tienen constructor que valida invariantes
- [ ] Errores son variables tipadas, no strings
- [ ] Funciones que cruzan capas reciben `context.Context`
- [ ] Repos están en `domain/` como interfaces, en `infrastructure/postgres/` como structs
- [ ] Tests unitarios para el dominio sin tocar DB

## Antipatrones prohibidos

❌ **"Service" gigante con todo adentro.** Un caso de uso = una struct = un método `Execute`.

❌ **Pasar `*sql.DB` o `*gin.Context` al dominio.** El dominio no conoce HTTP ni SQL.

❌ **Importar `infrastructure` desde `domain`.** Detectado en CI con [`go-cleanarch`](https://github.com/roblaszczak/go-cleanarch) o lint custom.

❌ **Usar `panic` en lugar de errores.** Salvo `init()` panics legítimos.

❌ **`init()` con efectos secundarios.** Especialmente conexiones a DB. Eso va en `wire.go` o `main.go`.

## Recursos del repo

Cuando construyas algo específico, consulta estos skills:
- `domain-modeling-quiniela` — cómo modelar entidades del dominio de quinielas
- `scoring-strategy` — cómo implementar las reglas de puntaje
- `bracket-prediction` — cómo manejar el agregado de bracket
- `sqlc-repository` — patrón para repositorios con sqlc
- `external-api-adapter` — patrón para integrar APIs externas
