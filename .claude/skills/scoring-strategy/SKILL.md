---
name: scoring-strategy
description: How to implement the points calculation engine for the Quiniela Mundial using the Strategy pattern. Use this skill whenever working on anything that calculates, recalculates, or stores points — match points, bracket points, ranking aggregation, score entries, or the scoring engine itself. Apply this for any code under internal/domain/scoring/ or internal/application/commands/compute*, and for any work that interacts with the score_entries table. Critical for keeping the scoring rules consistent with docs/SCORING-RULES.md.
---

# Scoring Strategy — Quiniela Mundial

Implementación del motor de cálculo de puntos aplicando **Strategy Pattern**.

**Fuente de verdad de las reglas:** `docs/SCORING-RULES.md`. Si las reglas en código y en el doc difieren, gana el doc — actualiza el código.

## Visión general

Hay dos tipos de pronóstico que dan puntos:
1. **MatchPrediction:** pronóstico de marcador → 0 a 5 pts por partido
2. **BracketPrediction:** pronóstico de avance del torneo → puntos por equipos que alcanzan cada fase

Cada uno se implementa como una `ScoringStrategy` separada. El `ScoringEngine` las orquesta.

## Interfaces

```go
// internal/domain/scoring/strategy.go
package scoring

type ScoringStrategy interface {
    // Compute calcula los ScoreEntries que produce esta estrategia
    // para un usuario dentro de un pool, dado el contexto.
    Compute(ctx context.Context, in ComputeInput) ([]ScoreEntry, error)
}

type ScoreEntry struct {
    UserID     user.ID
    PoolID     pool.ID
    SourceType SourceType
    SourceRef  string  // ej. "match:abc-123" o "bracket_stage:round_of_32:ARG"
    Points     int
    ComputedAt time.Time
}

type SourceType string

const (
    SourceMatch              SourceType = "match"
    SourceBracketStage       SourceType = "bracket_stage"
    SourceBracketThirdPlace  SourceType = "bracket_third_place"
    SourceBracketChampion    SourceType = "bracket_champion"
)
```

## MatchScoringStrategy

```go
// internal/domain/scoring/match_strategy.go
package scoring

type MatchScoringStrategy struct {
    predictions prediction.Repository
}

func NewMatchScoringStrategy(repo prediction.Repository) *MatchScoringStrategy {
    return &MatchScoringStrategy{predictions: repo}
}

type MatchComputeInput struct {
    Match  *match.Match
    UserID user.ID
    PoolID pool.ID
}

func (s *MatchScoringStrategy) ComputeForMatch(
    ctx context.Context,
    m *match.Match,
    pred *prediction.MatchPrediction,
    now time.Time,
) (ScoreEntry, error) {
    if m.Status() != match.StatusFinished {
        return ScoreEntry{}, ErrMatchNotFinished
    }

    points := 0
    homeReg, awayReg, err := m.RegularGoals()
    if err != nil {
        return ScoreEntry{}, err
    }

    // +1 por acertar goles del local (tiempo regular)
    if pred.HomeGoals() == homeReg {
        points += 1
    }
    // +1 por acertar goles del visitante (tiempo regular)
    if pred.AwayGoals() == awayReg {
        points += 1
    }
    // +3 por acertar ganador oficial (post-penales en knockout)
    actualWinner, err := m.OfficialWinner()
    if err != nil {
        return ScoreEntry{}, err
    }
    predWinner := match.WinnerFrom(pred.HomeGoals(), pred.AwayGoals())
    if predWinner == actualWinner {
        points += 3
    }

    return ScoreEntry{
        UserID:     pred.UserID(),
        PoolID:     pred.PoolID(),
        SourceType: SourceMatch,
        SourceRef:  fmt.Sprintf("match:%s", m.ID()),
        Points:     points,
        ComputedAt: now,
    }, nil
}
```

### Tabla de casos (debe testearse)

| Predicción | Tiempo regular | Penales | Stage | Puntos | Razón |
|---|---|---|---|---|---|
| 2-1 | 2-1 | — | grupo | 5 | Marcador exacto |
| 2-1 | 3-1 | — | grupo | 4 | Ganador + away |
| 2-1 | 2-0 | — | grupo | 4 | Ganador + home |
| 1-1 | 1-1 | — | grupo | 5 | Empate exacto |
| 2-1 | 0-1 | — | grupo | 0 | Perdedor opuesto |
| 1-1 | 1-1 | 5-4 | r32 | 2 | Empate goles regular, perdió "DRAW" |
| 2-1 | 2-1 | — | r32 | 5 | Ganador local, exacto |
| 2-1 | 1-2 | 5-3 | r32 | 3 | Pronosticó local, ganó local por penales tras empate distinto |
| 0-0 | 0-0 | — | grupo | 5 | Empate sin goles exacto |

> El cuarto-último caso es sutil: en r32 con predicción 2-1, tiempo regular fue 1-2 (away gana en regular), pero por penales ganó el local 5-3. El "ganador oficial" es HOME (local). Tu predicción 2-1 también señalaba HOME. Aciertas ganador (+3). No aciertas goles. Total: 3 pts.

## BracketScoringStrategy

```go
// internal/domain/scoring/bracket_strategy.go
package scoring

type BracketScoringStrategy struct{}

func NewBracketScoringStrategy() *BracketScoringStrategy {
    return &BracketScoringStrategy{}
}

// ComputeStagePoints calcula puntos por equipos que alcanzaron una fase específica.
// Se invoca cuando se completa esa fase.
func (s *BracketScoringStrategy) ComputeStagePoints(
    ctx context.Context,
    pred *prediction.BracketPrediction,
    stage tournament.Stage,
    actualTeamsAtStage []team.ID,
    now time.Time,
) []ScoreEntry {
    pointsPerTeam, predictedTeams := s.stageConfig(pred, stage)
    if pointsPerTeam == 0 {
        return nil
    }

    actualSet := setOf(actualTeamsAtStage)
    var entries []ScoreEntry
    for _, t := range predictedTeams {
        if actualSet[t] {
            entries = append(entries, ScoreEntry{
                UserID:     pred.UserID(),
                PoolID:     pred.PoolID(),
                SourceType: SourceBracketStage,
                SourceRef:  fmt.Sprintf("bracket:%s:%s", stage, t),
                Points:     pointsPerTeam,
                ComputedAt: now,
            })
        }
    }
    return entries
}

func (s *BracketScoringStrategy) stageConfig(
    pred *prediction.BracketPrediction,
    stage tournament.Stage,
) (int, []team.ID) {
    switch stage {
    case tournament.StageRoundOf32:
        return 3, pred.TeamsToRoundOf32()
    case tournament.StageRoundOf16:
        return 4, pred.TeamsToRoundOf16()
    case tournament.StageSemiFinal:
        return 5, pred.TeamsToSemiFinal()
    case tournament.StageFinal:
        return 10, pred.TeamsToFinal()
    }
    return 0, nil
}

func (s *BracketScoringStrategy) ComputeThirdPlace(
    pred *prediction.BracketPrediction,
    actualThirdPlace team.ID,
    now time.Time,
) *ScoreEntry {
    if pred.ThirdPlaceWinner() != actualThirdPlace {
        return nil
    }
    return &ScoreEntry{
        UserID:     pred.UserID(),
        PoolID:     pred.PoolID(),
        SourceType: SourceBracketThirdPlace,
        SourceRef:  "third_place",
        Points:     15,
        ComputedAt: now,
    }
}

func (s *BracketScoringStrategy) ComputeChampion(
    pred *prediction.BracketPrediction,
    actualChampion team.ID,
    now time.Time,
) *ScoreEntry {
    if pred.Champion() != actualChampion {
        return nil
    }
    return &ScoreEntry{
        UserID:     pred.UserID(),
        PoolID:     pred.PoolID(),
        SourceType: SourceBracketChampion,
        SourceRef:  "champion",
        Points:     20,
        ComputedAt: now,
    }
}

func setOf(teams []team.ID) map[team.ID]bool {
    m := make(map[team.ID]bool, len(teams))
    for _, t := range teams {
        m[t] = true
    }
    return m
}
```

## ScoringEngine (orquestador)

```go
// internal/application/commands/scoring_engine.go
package commands

type ScoringEngine struct {
    matchStrategy   *scoring.MatchScoringStrategy
    bracketStrategy *scoring.BracketScoringStrategy
    predictions     prediction.Repository
    brackets        prediction.BracketRepository
    scores          scoring.Repository
    pools           pool.Repository
    clock           Clock
}

// RecomputeMatch se invoca cuando un partido se finaliza.
func (e *ScoringEngine) RecomputeMatch(ctx context.Context, matchID match.ID) error {
    m, err := e.matches.FindByID(ctx, matchID)
    if err != nil {
        return err
    }
    if m.Status() != match.StatusFinished {
        return scoring.ErrMatchNotFinished
    }

    // Para cada pool, para cada usuario de ese pool, computar puntos
    pools, err := e.pools.FindAll(ctx)
    if err != nil {
        return err
    }
    now := e.clock.Now()

    for _, p := range pools {
        for _, member := range p.Members() {
            pred, err := e.predictions.FindByUserAndMatch(ctx, member.UserID, p.ID(), matchID)
            if errors.Is(err, prediction.ErrNotFound) {
                continue // usuario no pronosticó este partido
            }
            if err != nil {
                return err
            }
            entry, err := e.matchStrategy.ComputeForMatch(ctx, m, pred, now)
            if err != nil {
                return err
            }
            if err := e.scores.Upsert(ctx, entry); err != nil {
                return err
            }
        }
    }
    return nil
}

// RecomputeStage se invoca cuando una fase del torneo se completa.
func (e *ScoringEngine) RecomputeStage(
    ctx context.Context,
    stage tournament.Stage,
    actualTeams []team.ID,
) error {
    now := e.clock.Now()
    pools, _ := e.pools.FindAll(ctx)

    for _, p := range pools {
        for _, m := range p.Members() {
            bracket, err := e.brackets.FindByUserAndPool(ctx, m.UserID, p.ID())
            if errors.Is(err, prediction.ErrNotFound) {
                continue
            }
            if err != nil {
                return err
            }
            entries := e.bracketStrategy.ComputeStagePoints(ctx, bracket, stage, actualTeams, now)
            for _, entry := range entries {
                if err := e.scores.Upsert(ctx, entry); err != nil {
                    return err
                }
            }
        }
    }
    return nil
}
```

## Idempotencia (CRÍTICA)

El método `Upsert` del repo de scoring debe usar `(user_id, pool_id, source_type, source_ref)` como llave de conflicto:

```sql
-- queries/score_entries.sql
-- name: UpsertScoreEntry :exec
INSERT INTO score_entries (user_id, pool_id, source_type, source_ref, points, computed_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, pool_id, source_type, source_ref)
DO UPDATE SET
    points     = EXCLUDED.points,
    computed_at = EXCLUDED.computed_at;
```

```sql
-- migrations/0007_score_entries.up.sql
CREATE TABLE score_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    pool_id     UUID NOT NULL REFERENCES pools(id),
    source_type VARCHAR(50) NOT NULL,
    source_ref  VARCHAR(200) NOT NULL,
    points      INTEGER NOT NULL CHECK (points >= 0),
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, pool_id, source_type, source_ref)
);

CREATE INDEX idx_score_entries_pool_user ON score_entries(pool_id, user_id);
```

## Eventos que disparan recálculo

Suscríbete a estos eventos en `internal/interfaces/jobs/scoring_listener.go`:

| Evento | Handler |
|---|---|
| `MatchResultFinalized` | `engine.RecomputeMatch(matchID)` |
| `GroupStageCompleted` | `engine.RecomputeStage(StageRoundOf32, teams)` |
| `RoundOfThirtyTwoCompleted` | `engine.RecomputeStage(StageRoundOf16, teams)` |
| `RoundOfSixteenCompleted` | `engine.RecomputeStage(StageSemiFinal, teams)` |
| `QuarterFinalsCompleted` | `engine.RecomputeStage(StageFinal, teams)` |
| `ThirdPlaceMatchFinalized` | `engine.RecomputeThirdPlace(teamID)` |
| `FinalMatchFinalized` | `engine.RecomputeChampion(teamID)` |

## Tests obligatorios

Cubrir como mínimo:

```go
func TestMatchStrategy_ExactScore(t *testing.T) {
    // pred 2-1, result 2-1 → 5 pts
}

func TestMatchStrategy_WinnerOnly(t *testing.T) {
    // pred 2-1, result 3-0 → 3 pts (ganador, 0 goles)
}

func TestMatchStrategy_KnockoutDrawWonByPenalties(t *testing.T) {
    // pred 1-1, result 1-1 + penales 5-4 (HOME) en r32 → 2 pts
}

func TestBracketStrategy_AllRoundOf32Correct(t *testing.T) {
    // 32 equipos predichos = 32 reales → 96 pts (32 * 3)
}

func TestBracketStrategy_ChampionCorrect(t *testing.T) {
    // pred.Champion == actualChampion → 20 pts
}

func TestScoringEngine_RecomputeIsIdempotent(t *testing.T) {
    // Llamar RecomputeMatch dos veces no duplica entries
}
```

## Antipatrones

❌ **Calcular puntos en el handler HTTP.** El cálculo es lógica de dominio, debe estar en `internal/domain/scoring/`.

❌ **Insertar `ScoreEntry` directamente con INSERT.** Siempre UPSERT con la llave única.

❌ **Calcular ranking en tiempo real con queries complejas en cada pedido.** Si el ranking se vuelve lento, materializar en una vista `pool_rankings` que se refresca con los eventos.

❌ **Hardcodear los valores 3/4/5/10/15/20 en múltiples lugares.** Estos viven en `tournament.StageDefinition` o en constantes del paquete `scoring`.
