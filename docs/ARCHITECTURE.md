# Arquitectura

Sistema construido siguiendo **Clean Architecture** (Robert C. Martin) con influencias de **Hexagonal Architecture** (Alistair Cockburn) y **DDD táctico** (Eric Evans).

## Principios rectores

1. **Regla de la dependencia:** las capas externas dependen de las internas. Nunca al revés.
2. **El dominio no sabe que existe HTTP, SQL, ni APIs externas.** Solo conoce sus reglas de negocio.
3. **Inversión de dependencias:** las capas internas declaran interfaces, las externas las implementan.
4. **Casos de uso explícitos:** cada acción del sistema es un caso de uso con un único método `Execute`.

## Las cuatro capas

```
┌──────────────────────────────────────────────────────────────┐
│  INTERFACES (HTTP handlers, CLI, cron jobs)                  │
│  ↓ usa                                                       │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  APPLICATION (Use Cases / Interactors)                 │  │
│  │  ↓ usa                                                 │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │  DOMAIN (Entidades, Value Objects, Servicios)    │  │  │
│  │  │  - Define interfaces (Ports)                     │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  INFRASTRUCTURE (Adapters: PostgreSQL, APIs externas, SMTP)  │
│  ↑ implementa las interfaces declaradas en DOMAIN            │
└──────────────────────────────────────────────────────────────┘
```

### 1. Domain (`internal/domain/`)

La capa más interna. **Cero dependencias externas**, solo stdlib de Go.

Contiene:
- **Entidades:** objetos con identidad (`User`, `Pool`, `Match`, `MatchPrediction`, `BracketPrediction`)
- **Value Objects:** objetos sin identidad, inmutables (`Score`, `TeamID`, `Email`, `Points`)
- **Servicios de dominio:** lógica que no pertenece a una sola entidad (ej. `BracketCoherenceValidator`)
- **Ports (interfaces):** contratos que la infraestructura debe implementar

Ejemplo:
```go
// internal/domain/prediction/match_prediction.go
package prediction

type MatchPrediction struct {
    id        ID
    userID    user.ID
    poolID    pool.ID
    matchID   match.ID
    homeGoals int
    awayGoals int
}

func NewMatchPrediction(...) (*MatchPrediction, error) {
    // Validaciones de invariantes
}

// Port: la infraestructura debe implementar esto
type Repository interface {
    Save(ctx context.Context, p *MatchPrediction) error
    FindByUserAndMatch(ctx context.Context, userID user.ID, matchID match.ID) (*MatchPrediction, error)
}
```

### 2. Application (`internal/application/`)

Casos de uso del sistema. Orquesta entidades y servicios de dominio para cumplir una acción.

Separación CQRS ligera:
- `commands/` — operaciones que modifican estado (crear pronóstico, cargar resultado)
- `queries/` — operaciones de lectura (ranking, listar pronósticos)

Ejemplo:
```go
// internal/application/commands/submit_match_prediction.go
package commands

type SubmitMatchPrediction struct {
    predictions prediction.Repository
    matches     match.Repository
    clock       Clock
}

type SubmitMatchPredictionInput struct {
    UserID    string
    PoolID    string
    MatchID   string
    HomeGoals int
    AwayGoals int
}

func (uc *SubmitMatchPrediction) Execute(ctx context.Context, in SubmitMatchPredictionInput) error {
    m, err := uc.matches.FindByID(ctx, match.ID(in.MatchID))
    if err != nil {
        return err
    }
    if uc.clock.Now().After(m.KickoffAt()) {
        return prediction.ErrPredictionClosed
    }
    p, err := prediction.NewMatchPrediction(...)
    if err != nil {
        return err
    }
    return uc.predictions.Save(ctx, p)
}
```

### 3. Infrastructure (`internal/infrastructure/`)

Implementaciones concretas de los ports definidos en `domain/`.

- `persistence/postgres/` — repositorios SQL con `sqlc` generado
- `external/openfootball/` — adapter para JSON del Mundial
- `external/footballdata/` — adapter para football-data.org API
- `external/balldontlie/` — adapter alternativo
- `auth/jwt/` — generación/validación de tokens
- `email/smtp/` o `email/resend/` — envío de invitaciones

Ejemplo:
```go
// internal/infrastructure/persistence/postgres/match_prediction_repo.go
package postgres

type MatchPredictionRepo struct {
    queries *sqlc.Queries
}

func (r *MatchPredictionRepo) Save(ctx context.Context, p *prediction.MatchPrediction) error {
    return r.queries.UpsertMatchPrediction(ctx, sqlc.UpsertMatchPredictionParams{
        ID:        p.ID().String(),
        UserID:    p.UserID().String(),
        // ...
    })
}
```

### 4. Interfaces (`internal/interfaces/`)

Adapters de entrada — cómo el mundo exterior habla con el sistema.

- `http/` — handlers Gin, routing, middleware
- `jobs/` — cron jobs (sync de resultados, recálculo de rankings)
- `cli/` — comandos (seed, recalculate scores)

Ejemplo:
```go
// internal/interfaces/http/handlers/predictions.go
package handlers

type PredictionsHandler struct {
    submitMatch *commands.SubmitMatchPrediction
}

func (h *PredictionsHandler) Submit(c *gin.Context) {
    var req SubmitMatchRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    err := h.submitMatch.Execute(c.Request.Context(), commands.SubmitMatchPredictionInput{
        UserID:    c.GetString("user_id"),
        PoolID:    req.PoolID,
        MatchID:   req.MatchID,
        HomeGoals: req.HomeGoals,
        AwayGoals: req.AwayGoals,
    })
    if err != nil {
        // mapeo de errores de dominio a HTTP
        if errors.Is(err, prediction.ErrPredictionClosed) {
            c.JSON(409, gin.H{"error": "prediction window closed"})
            return
        }
        c.JSON(500, gin.H{"error": "internal error"})
        return
    }
    c.JSON(201, gin.H{"status": "ok"})
}
```

## Patrones de diseño aplicados

### Estructurales

- **Repository:** abstracción de persistencia. Cada agregado tiene su repo.
- **Adapter:** para integrar APIs externas heterogéneas detrás de una interfaz común (`MatchResultProvider`).
- **Composite (ligero):** el `ScoringEngine` compone múltiples `ScoringStrategy`.

### Comportamentales

- **Strategy:** reglas de puntaje intercambiables. `MatchScoringStrategy` y `BracketScoringStrategy` implementan la misma interfaz `ScoringStrategy`.
- **Use Case (Interactor):** cada acción de la aplicación es un struct con `Execute`.
- **Observer / Event-driven (ligero):** cuando se finaliza un resultado, se publica un evento `MatchResultFinalized` que dispara recálculos.

### Creacionales

- **Functional Options:** para constructores con muchos parámetros opcionales.
- **Dependency Injection con `wire`** (compile-time, sin reflection): el grafo de dependencias se arma en `cmd/api/wire.go`.

### Modernos en Go

- **`context.Context` en todas las firmas** que cruzan capas (cancelación, deadline, trazabilidad).
- **Errors as values + `errors.Is`/`errors.As`:** errores de dominio como variables, no como strings.
- **Generics donde tienen sentido** (ej. `Result[T any]` para queries que pueden fallar).

## Estructura de carpetas final

```
quiniela-mundial/
├── cmd/
│   └── api/
│       ├── main.go              # Entry point
│       └── wire.go              # Wiring de dependencias
├── internal/
│   ├── domain/
│   │   ├── user/
│   │   ├── pool/
│   │   ├── tournament/
│   │   ├── team/
│   │   ├── match/
│   │   ├── prediction/          # Match + Bracket
│   │   ├── scoring/             # Strategies + Engine
│   │   └── shared/              # IDs, errores comunes
│   ├── application/
│   │   ├── commands/
│   │   └── queries/
│   ├── infrastructure/
│   │   ├── persistence/postgres/
│   │   │   ├── sqlc/            # Generado por sqlc
│   │   │   └── *_repo.go        # Adapters
│   │   ├── external/
│   │   │   ├── openfootball/
│   │   │   ├── footballdata/
│   │   │   └── balldontlie/
│   │   ├── auth/jwt/
│   │   ├── email/
│   │   └── eventbus/
│   └── interfaces/
│       ├── http/
│       │   ├── handlers/
│       │   ├── middleware/
│       │   └── router.go
│       ├── jobs/
│       └── cli/
├── migrations/                  # Archivos .up.sql / .down.sql
├── queries/                     # Archivos .sql para sqlc
├── api/openapi.yaml             # Spec OpenAPI 3
├── frontend/                    # Angular standalone
└── docs/
```

## Diagrama de dependencias

```
cmd/api ──→ interfaces ──→ application ──→ domain ←── infrastructure
              ↑                                            │
              └────────── infrastructure ──────────────────┘
                    (handlers reciben repos concretos
                     que implementan ports del domain)
```

Las flechas son la dirección de dependencia (quién importa a quién). Nota que `infrastructure` "apunta hacia adentro" implementando ports del `domain`, pero `domain` no importa nada de `infrastructure`.

## Testing por capa

| Capa | Tipo de test | Herramienta |
|------|--------------|-------------|
| Domain | Unitarios puros | `testing` + `testify` |
| Application | Unitarios con mocks | `gomock` o stubs manuales |
| Infrastructure (repos) | Integración con DB | `testcontainers-go` |
| Infrastructure (APIs externas) | Mocks de HTTP | `httptest` |
| Interfaces (handlers) | Tests de contrato | `httptest` + json fixtures |
| End-to-end | Pocos, críticos | Docker compose + Postman/Bruno |

## Lecturas recomendadas

- *Clean Architecture* — Robert C. Martin
- *Domain-Driven Design Distilled* — Vaughn Vernon
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout) (con criterio, no dogmáticamente)
- [Go Clean Architecture by Bxcodec](https://github.com/bxcodec/go-clean-arch) (referencia práctica)
