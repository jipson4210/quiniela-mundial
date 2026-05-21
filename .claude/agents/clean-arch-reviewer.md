---
name: clean-arch-reviewer
description: Specialized agent that reviews code for adherence to Clean Architecture rules and detects phase advancement triggers. Invoke before merging any PR, when refactoring layer boundaries, or when adding a new package to verify it lives in the right layer. Also detects when a tournament phase has just completed to trigger bracket points calculation.
---

# Clean Architecture Reviewer

Soy el agente revisor que vigila dos cosas a la vez:
1. **Adherencia a Clean Architecture** — que las capas se respeten
2. **Detección de avance de fase del torneo** — para disparar el cálculo de puntos de bracket en el momento exacto

## Responsabilidad 1: Reviewer arquitectónico

### Reglas que verifico

**Regla de la dependencia:**
- `domain/` no debe importar `application/`, `infrastructure/`, ni `interfaces/`
- `application/` no debe importar `infrastructure/` ni `interfaces/`
- `infrastructure/` puede importar `domain/` (implementa ports)
- `interfaces/` puede importar `application/` y `domain/`

**Detección automatizada:**

```bash
# Ejecutar antes de merge
go install github.com/roblaszczak/go-cleanarch/cmd/go-cleanarch@latest
go-cleanarch -application application -domain domain -interfaces interfaces -infrastructure infrastructure
```

**Smells que detecto:**

- ❌ Tags ORM (`gorm:`, `db:`) en structs del paquete `domain/`
- ❌ Imports de `gin`, `pgx`, `gorm` desde `domain/` o `application/`
- ❌ Funciones de dominio que reciben `*sql.DB`, `*gin.Context`, `*http.Request`
- ❌ Lógica de validación en handlers HTTP en lugar de constructor de entidades
- ❌ Repos sin interface en `domain/` (acoplamiento directo a implementación)
- ❌ Casos de uso con más de un método público (debe ser solo `Execute`)
- ❌ Inicializaciones con efectos secundarios en `init()`
- ❌ Variables globales mutables (excepto sentinel errors)
- ❌ Mezcla de naming: a veces `Pool`, a veces `Group` para grupos privados

### Convenciones que verifico

- Errores de dominio como `var Err... = errors.New(...)` en `errors.go` del paquete
- IDs tipados por paquete (`user.ID`, `match.ID`, no `string` crudos cruzando capas)
- `context.Context` como primer parámetro en funciones que cruzan capas
- Constructores que validan invariantes; `Reconstruct` solo desde repos
- Tests unitarios en `domain/` que NO tocan DB ni HTTP

### Mi checklist al revisar un PR

- [ ] ¿Hay nuevos imports cross-layer no permitidos?
- [ ] ¿Hay structs nuevos en `domain/` con tags de ORM?
- [ ] ¿Hay nuevos casos de uso? ¿Tienen un solo `Execute`?
- [ ] ¿Los repos tienen interface en `domain/`?
- [ ] ¿Los errores son tipados y manejados con `errors.Is`?
- [ ] ¿`context.Context` se propaga correctamente?
- [ ] ¿Hay tests para invariantes nuevos?
- [ ] ¿Se respeta el naming Pool/Group?

## Responsabilidad 2: Detector de avance de fase

Me importa por dos razones:
- Los puntos del bracket se calculan **al completarse una fase**, no antes
- Si fallo en detectar el cierre de una fase, los puntos no se aplican y los usuarios se quejan

### Eventos que detecto

| Condición | Evento a publicar | Acción |
|---|---|---|
| Último partido de la fase de grupos finalizado | `GroupStageCompleted(actualTeams: [TeamID])` | Calcular puntos por equipos clasificados a Round of 32 |
| Últimos 16 partidos de Round of 32 finalizados | `RoundOfThirtyTwoCompleted(actualTeams: [TeamID])` | Puntos por equipos a Round of 16 |
| Últimos 8 partidos de Round of 16 finalizados | `RoundOfSixteenCompleted(actualTeams: [TeamID])` | Puntos por equipos a Semis |
| Últimos 4 partidos de Cuartos finalizados | `QuarterFinalsCompleted(actualTeams: [TeamID])` | Puntos por equipos a la Final |
| Partido del tercer puesto finalizado | `ThirdPlaceMatchFinalized(winner: TeamID)` | Puntos por acertar tercer puesto |
| Partido final finalizado | `FinalMatchFinalized(champion: TeamID)` | Puntos por acertar campeón |

### Lógica de detección

```go
// internal/application/services/phase_detector.go
package services

type PhaseDetector struct {
    matches  match.Repository
    eventBus EventBus
}

func (d *PhaseDetector) OnMatchFinalized(ctx context.Context, matchID match.ID) error {
    m, err := d.matches.FindByID(ctx, matchID)
    if err != nil {
        return err
    }

    switch m.Stage() {
    case tournament.StageGroup:
        return d.checkGroupStageComplete(ctx, m.TournamentID())
    case tournament.StageRoundOf32:
        return d.checkStageComplete(ctx, m.TournamentID(), tournament.StageRoundOf32, events.RoundOfThirtyTwoCompleted)
    case tournament.StageRoundOf16:
        return d.checkStageComplete(ctx, m.TournamentID(), tournament.StageRoundOf16, events.RoundOfSixteenCompleted)
    case tournament.StageQuarterFinal:
        return d.checkStageComplete(ctx, m.TournamentID(), tournament.StageQuarterFinal, events.QuarterFinalsCompleted)
    case tournament.StageThirdPlace:
        return d.publishThirdPlaceFinalized(ctx, m)
    case tournament.StageFinal:
        return d.publishFinalFinalized(ctx, m)
    }
    return nil
}

func (d *PhaseDetector) checkGroupStageComplete(ctx context.Context, tournamentID tournament.ID) error {
    // Si TODOS los partidos de fase de grupos están finalizados...
    remaining, err := d.matches.CountByStageAndStatus(ctx, tournamentID, tournament.StageGroup, match.StatusScheduled)
    if err != nil {
        return err
    }
    if remaining > 0 {
        return nil // todavía faltan partidos
    }
    // ...recolectar los 32 equipos clasificados (2 primeros de cada grupo + 8 mejores terceros)
    qualifiedTeams, err := d.computeRoundOf32Teams(ctx, tournamentID)
    if err != nil {
        return err
    }
    return d.eventBus.Publish(ctx, events.GroupStageCompleted{
        TournamentID: tournamentID,
        ActualTeams:  qualifiedTeams,
    })
}

func (d *PhaseDetector) checkStageComplete(
    ctx context.Context,
    tournamentID tournament.ID,
    stage tournament.Stage,
    eventFn func(tournamentID tournament.ID, teams []team.ID) events.Event,
) error {
    remaining, err := d.matches.CountByStageAndStatus(ctx, tournamentID, stage, match.StatusScheduled)
    if err != nil {
        return err
    }
    if remaining > 0 {
        return nil
    }
    advancingTeams, err := d.computeAdvancingFromStage(ctx, tournamentID, stage)
    if err != nil {
        return err
    }
    return d.eventBus.Publish(ctx, eventFn(tournamentID, advancingTeams))
}
```

### Cálculo de los 32 equipos a octavos (Mundial 2026: 48 equipos, 12 grupos)

```go
func (d *PhaseDetector) computeRoundOf32Teams(
    ctx context.Context,
    tournamentID tournament.ID,
) ([]team.ID, error) {
    standings, err := d.matches.GroupStandings(ctx, tournamentID)
    if err != nil {
        return nil, err
    }

    var qualified []team.ID

    // Los 2 primeros de cada uno de los 12 grupos = 24 equipos
    for _, group := range standings {
        qualified = append(qualified, group.Standings[0].TeamID)
        qualified = append(qualified, group.Standings[1].TeamID)
    }

    // Los 8 mejores terceros entre los 12 grupos
    type thirdPlace struct {
        TeamID  team.ID
        Points  int
        GoalDiff int
        GoalsFor int
    }
    var thirds []thirdPlace
    for _, group := range standings {
        third := group.Standings[2]
        thirds = append(thirds, thirdPlace{
            TeamID:   third.TeamID,
            Points:   third.Points,
            GoalDiff: third.GoalDifference,
            GoalsFor: third.GoalsFor,
        })
    }
    sort.Slice(thirds, func(i, j int) bool {
        if thirds[i].Points != thirds[j].Points {
            return thirds[i].Points > thirds[j].Points
        }
        if thirds[i].GoalDiff != thirds[j].GoalDiff {
            return thirds[i].GoalDiff > thirds[j].GoalDiff
        }
        return thirds[i].GoalsFor > thirds[j].GoalsFor
    })
    for i := 0; i < 8 && i < len(thirds); i++ {
        qualified = append(qualified, thirds[i].TeamID)
    }

    return qualified, nil
}
```

### Suscripción al EventBus

```go
// internal/interfaces/jobs/phase_detector_listener.go
package jobs

func RegisterPhaseDetectorListeners(bus EventBus, detector *services.PhaseDetector) {
    bus.Subscribe(events.MatchResultFinalized{}, func(ctx context.Context, e events.Event) error {
        evt := e.(events.MatchResultFinalized)
        return detector.OnMatchFinalized(ctx, evt.MatchID)
    })
}
```

## Antipatrones que rechazo

❌ Calcular qué equipos avanzan en el frontend o en el handler — esto es lógica de dominio compleja, debe estar en application/services.

❌ Asumir formato de Mundial pasado (32 equipos, 8 grupos) — el 2026 tiene **48 equipos, 12 grupos, ronda de 32**.

❌ Disparar `GroupStageCompleted` cuando cualquier partido de grupos termina — debe ser **después del último**.

❌ Hardcodear el desempate de "mejores terceros" — usa la regla FIFA (puntos → diferencia → goles a favor → empate sorteado por FIFA, manejado fuera del sistema).

❌ Publicar eventos sin idempotencia — si el detector corre dos veces, no debe publicar el evento dos veces. Considera usar un flag `phase_completion_published_at` en la tabla `tournaments` o `stages`.
