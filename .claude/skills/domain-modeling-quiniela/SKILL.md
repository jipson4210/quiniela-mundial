---
name: domain-modeling-quiniela
description: How to model domain entities, value objects, and aggregates specific to the Quiniela Mundial project. Use this skill when creating or modifying any domain object related to pools (private groups), users, tournament structure, matches, predictions, or pool memberships. Apply this whenever the task touches anything in /internal/domain/, even if the user just says "add a field" or "create an entity" without mentioning DDD or domain modeling explicitly. Critical for keeping the domain layer pure and invariants enforced.
---

# Domain Modeling — Quiniela Mundial

Guía para modelar entidades del dominio específico de este proyecto. Lee también `docs/DOMAIN-MODEL.md` para la especificación completa.

## Agregados existentes

| Agregado | Paquete | Responsabilidad |
|----------|---------|----------------|
| `User` | `internal/domain/user/` | Identidad |
| `Pool` | `internal/domain/pool/` | Grupo privado + miembros + invitaciones |
| `Tournament` | `internal/domain/tournament/` | Estructura del Mundial |
| `Team` | `internal/domain/team/` | Selecciones nacionales |
| `Match` | `internal/domain/match/` | Partidos individuales |
| `MatchPrediction` | `internal/domain/prediction/` | Pronóstico de marcador |
| `BracketPrediction` | `internal/domain/prediction/` | Pronóstico de avance del torneo |

## Distinción crítica: Pool vs Group

- **Pool** = grupo privado de quiniela (ej. "Familia Macías", "Oficina DTIC").
- **Group** = grupo del Mundial (Grupo A, B, ..., L).

Nunca usar "Group" para grupos de usuarios. Siempre "Pool". Aplicar este uso en:
- Nombres de structs
- Nombres de tablas SQL
- URLs de la API (`/pools/...`, no `/groups/...`)
- UI Angular (`PoolListComponent`, no `GroupListComponent`)

## Identificadores

Usar **UUID v7** (sortable). En Go:

```go
// internal/domain/shared/ids.go
package shared

import "github.com/google/uuid"

type ID string

func NewID() ID {
    id, _ := uuid.NewV7()
    return ID(id.String())
}
```

Luego cada paquete tiene su propio tipo:

```go
// internal/domain/user/user.go
package user

import "github.com/<repo>/internal/domain/shared"

type ID shared.ID
```

Esto da seguridad de tipos (no puedes pasar un `match.ID` a una función que espera `user.ID`).

## Value Objects clave

### Email

```go
// internal/domain/shared/email.go
package shared

type Email string

func NewEmail(s string) (Email, error) {
    s = strings.TrimSpace(strings.ToLower(s))
    if !emailRegex.MatchString(s) {
        return "", ErrInvalidEmail
    }
    return Email(s), nil
}

func (e Email) String() string { return string(e) }
```

### Score (marcador de partido)

```go
// internal/domain/match/score.go
package match

type Score struct {
    home int
    away int
}

func NewScore(home, away int) (Score, error) {
    if home < 0 || away < 0 {
        return Score{}, ErrNegativeScore
    }
    if home > 30 || away > 30 {
        return Score{}, ErrUnreasonableScore
    }
    return Score{home: home, away: away}, nil
}

func (s Score) Home() int { return s.home }
func (s Score) Away() int { return s.away }

func (s Score) Winner() Winner {
    switch {
    case s.home > s.away: return WinnerHome
    case s.home < s.away: return WinnerAway
    default: return WinnerDraw
    }
}
```

### Stage

```go
// internal/domain/tournament/stage.go
package tournament

type Stage string

const (
    StageGroup        Stage = "group"
    StageRoundOf32    Stage = "round_of_32"   // Octavos en formato 48 equipos
    StageRoundOf16    Stage = "round_of_16"   // Ronda de 16
    StageQuarterFinal Stage = "quarter_final"
    StageSemiFinal    Stage = "semi_final"
    StageThirdPlace   Stage = "third_place"
    StageFinal        Stage = "final"
)

func (s Stage) IsKnockout() bool {
    return s != StageGroup
}
```

## Entidad Pool (compleja)

`Pool` es un **agregado** que contiene `PoolMember`s. No expongas `PoolMember` fuera del agregado.

```go
// internal/domain/pool/pool.go
package pool

type Pool struct {
    id           ID
    name         string
    description  string
    creatorID    user.ID
    tournamentID tournament.ID
    settings     Settings
    members      []Member
    createdAt    time.Time
}

type Member struct {
    UserID    user.ID
    Role      Role
    JoinedAt  time.Time
    InvitedBy *user.ID
}

type Role string

const (
    RoleCreator Role = "creator"
    RoleAdmin   Role = "admin"
    RoleMember  Role = "member"
)

type Settings struct {
    MatchPredictionCutoffMinutes int
    ExtraTimeRule                ExtraTimeRule
    ShowOtherPredictions         bool
}

// Constructor
func NewPool(creatorID user.ID, tournamentID tournament.ID, name, description string, now time.Time) (*Pool, error) {
    if len(name) < 3 || len(name) > 80 {
        return nil, ErrInvalidName
    }
    p := &Pool{
        id:           ID(shared.NewID()),
        name:         name,
        description:  description,
        creatorID:    creatorID,
        tournamentID: tournamentID,
        settings:     defaultSettings(),
        createdAt:    now,
    }
    p.members = []Member{{
        UserID:   creatorID,
        Role:     RoleCreator,
        JoinedAt: now,
    }}
    return p, nil
}

// Operaciones de dominio (mutaciones controladas)
func (p *Pool) PromoteToAdmin(actor user.ID, target user.ID) error {
    if !p.isCreator(actor) {
        return ErrUnauthorized
    }
    for i, m := range p.members {
        if m.UserID == target {
            if m.Role == RoleAdmin {
                return ErrAlreadyAdmin
            }
            p.members[i].Role = RoleAdmin
            return nil
        }
    }
    return ErrMemberNotFound
}

func (p *Pool) AddMember(userID user.ID, invitedBy user.ID, now time.Time) error {
    for _, m := range p.members {
        if m.UserID == userID {
            return ErrAlreadyMember
        }
    }
    p.members = append(p.members, Member{
        UserID:    userID,
        Role:      RoleMember,
        JoinedAt:  now,
        InvitedBy: &invitedBy,
    })
    return nil
}

func (p *Pool) CanInvite(actor user.ID) bool {
    for _, m := range p.members {
        if m.UserID == actor {
            return m.Role == RoleCreator || m.Role == RoleAdmin
        }
    }
    return false
}

// Getters
func (p *Pool) ID() ID                        { return p.id }
func (p *Pool) Name() string                  { return p.name }
func (p *Pool) Members() []Member             { return append([]Member{}, p.members...) } // copia defensiva
```

## Entidad Match con resultado especial

`Match` debe manejar la complejidad del resultado post-penales:

```go
// internal/domain/match/match.go
package match

type Match struct {
    id           ID
    tournamentID tournament.ID
    stage        tournament.Stage
    groupID      *GroupID
    homeTeamID   team.ID
    awayTeamID   team.ID
    kickoffAt    time.Time
    venue        string
    status       Status
    result       *Result
}

type Status string

const (
    StatusScheduled  Status = "scheduled"
    StatusInProgress Status = "in_progress"
    StatusFinished   Status = "finished"
    StatusCancelled  Status = "cancelled"
)

type Result struct {
    HomeGoalsRegular        int
    AwayGoalsRegular        int
    HomeGoalsAfterET        *int
    AwayGoalsAfterET        *int
    HomeGoalsAfterPenalties *int
    AwayGoalsAfterPenalties *int
    FinalizedAt             time.Time
    Source                  ResultSource
}

// OfficialWinner devuelve el ganador "oficial" según las reglas:
// - Fase de grupos: solo tiempo regular
// - Eliminación directa: post-penales si los hubo
func (m *Match) OfficialWinner() (Winner, error) {
    if m.result == nil {
        return "", ErrNoResult
    }
    if !m.stage.IsKnockout() {
        return winnerFrom(m.result.HomeGoalsRegular, m.result.AwayGoalsRegular), nil
    }
    if m.result.HomeGoalsAfterPenalties != nil {
        return winnerFrom(*m.result.HomeGoalsAfterPenalties, *m.result.AwayGoalsAfterPenalties), nil
    }
    if m.result.HomeGoalsAfterET != nil {
        return winnerFrom(*m.result.HomeGoalsAfterET, *m.result.AwayGoalsAfterET), nil
    }
    return winnerFrom(m.result.HomeGoalsRegular, m.result.AwayGoalsRegular), nil
}

// RegularGoals devuelve siempre los goles del tiempo regular,
// que es lo que el usuario pronosticó.
func (m *Match) RegularGoals() (home, away int, err error) {
    if m.result == nil {
        return 0, 0, ErrNoResult
    }
    return m.result.HomeGoalsRegular, m.result.AwayGoalsRegular, nil
}
```

## Tests obligatorios para entidades del dominio

Crear tests unitarios sin DB:

```go
// internal/domain/pool/pool_test.go
package pool_test

func TestNewPool_ValidName(t *testing.T) {
    p, err := pool.NewPool(creatorID, tournamentID, "Familia Macías", "", time.Now())
    require.NoError(t, err)
    assert.Equal(t, "Familia Macías", p.Name())
    assert.Len(t, p.Members(), 1)
    assert.Equal(t, pool.RoleCreator, p.Members()[0].Role)
}

func TestNewPool_TooShortName(t *testing.T) {
    _, err := pool.NewPool(creatorID, tournamentID, "ab", "", time.Now())
    assert.ErrorIs(t, err, pool.ErrInvalidName)
}

func TestPool_PromoteToAdmin_NonCreatorFails(t *testing.T) {
    p, _ := pool.NewPool(creatorID, tournamentID, "Test", "", time.Now())
    _ = p.AddMember(otherUserID, creatorID, time.Now())
    err := p.PromoteToAdmin(otherUserID, otherUserID) // se promueve a sí mismo
    assert.ErrorIs(t, err, pool.ErrUnauthorized)
}
```

## Errores comunes a evitar

❌ **Exponer slices internos sin copia defensiva.** Si retornas `p.members` directamente, el caller puede mutar el agregado.

❌ **Setters públicos.** Mutaciones siempre vía operaciones de dominio (`PromoteToAdmin`, `AddMember`), no `SetMembers([]Member)`.

❌ **Lógica de validación dispersa.** Si una entidad solo se crea/modifica vía sus métodos, no necesitas validar en los handlers ni casos de uso.

❌ **Convertir entre IDs de paquetes distintos sin pasar por strings.** Si tienes `user.ID` y necesitas pasar a otra capa, conviértela a `string` en el adapter, no en el dominio.
