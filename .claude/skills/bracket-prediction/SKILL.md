---
name: bracket-prediction
description: How to implement the BracketPrediction aggregate — the user's prediction of which teams advance through each stage of the World Cup. Use this skill when creating, updating, or validating bracket predictions, when enforcing the coherence invariant (champion must be in finalists, finalists in semifinalists, etc.), or when freezing brackets at tournament kickoff. Apply this whenever working on /internal/domain/prediction/bracket*, on the Angular bracket form component, or on any code that touches the teams_to_* fields. This is the most complex piece of the domain.
---

# Bracket Prediction Aggregate

El `BracketPrediction` es el pronóstico más complejo del sistema. Es **una sola entidad por (user, pool)** que predice el avance completo del torneo.

## Estructura

```go
// internal/domain/prediction/bracket.go
package prediction

type BracketPrediction struct {
    id                  ID
    userID              user.ID
    poolID              pool.ID
    tournamentID        tournament.ID
    teamsToRoundOf32    []team.ID  // 32 equipos
    teamsToRoundOf16    []team.ID  // 16
    teamsToQuarterFinal []team.ID  // 8
    teamsToSemiFinal    []team.ID  // 4
    teamsToFinal        []team.ID  // 2
    thirdPlaceWinner    team.ID
    champion            team.ID
    submittedAt         time.Time
    updatedAt           time.Time
}
```

> **Sobre los slices:** internamente usar `map[team.ID]bool` para validaciones rápidas, pero la API externa (constructor, getters) usa slices ordenados o sets explícitos según convenga.

## Invariante de coherencia (regla más importante)

La jerarquía debe respetarse: **el champion sale de los finalistas, los finalistas de los semifinalistas, etc.**

```go
// internal/domain/prediction/bracket_validator.go
package prediction

type BracketValidator struct{}

func (v *BracketValidator) Validate(b *BracketPrediction) error {
    if err := v.validateSizes(b); err != nil {
        return err
    }
    if err := v.validateUniqueness(b); err != nil {
        return err
    }
    if err := v.validateHierarchy(b); err != nil {
        return err
    }
    if err := v.validateThirdPlace(b); err != nil {
        return err
    }
    if err := v.validateChampion(b); err != nil {
        return err
    }
    return nil
}

func (v *BracketValidator) validateSizes(b *BracketPrediction) error {
    if len(b.teamsToRoundOf32) != 32 {
        return fmt.Errorf("%w: expected 32 teams in round of 32, got %d",
            ErrInvalidBracketSize, len(b.teamsToRoundOf32))
    }
    if len(b.teamsToRoundOf16) != 16 {
        return fmt.Errorf("%w: round of 16", ErrInvalidBracketSize)
    }
    if len(b.teamsToQuarterFinal) != 8 {
        return fmt.Errorf("%w: quarter final", ErrInvalidBracketSize)
    }
    if len(b.teamsToSemiFinal) != 4 {
        return fmt.Errorf("%w: semi final", ErrInvalidBracketSize)
    }
    if len(b.teamsToFinal) != 2 {
        return fmt.Errorf("%w: final", ErrInvalidBracketSize)
    }
    return nil
}

func (v *BracketValidator) validateUniqueness(b *BracketPrediction) error {
    // Cada slice no debe tener equipos repetidos
    if !isUnique(b.teamsToRoundOf32) {
        return fmt.Errorf("%w: round of 32 has duplicates", ErrBracketDuplicate)
    }
    // ... etc
    return nil
}

func (v *BracketValidator) validateHierarchy(b *BracketPrediction) error {
    // teamsToRoundOf16 ⊂ teamsToRoundOf32
    r32 := setOf(b.teamsToRoundOf32)
    for _, t := range b.teamsToRoundOf16 {
        if !r32[t] {
            return fmt.Errorf("%w: team %s in round of 16 but not in round of 32",
                ErrBracketIncoherent, t)
        }
    }

    // teamsToQuarterFinal ⊂ teamsToRoundOf16
    r16 := setOf(b.teamsToRoundOf16)
    for _, t := range b.teamsToQuarterFinal {
        if !r16[t] {
            return fmt.Errorf("%w: team %s in quarter final but not in round of 16",
                ErrBracketIncoherent, t)
        }
    }

    // teamsToSemiFinal ⊂ teamsToQuarterFinal
    qf := setOf(b.teamsToQuarterFinal)
    for _, t := range b.teamsToSemiFinal {
        if !qf[t] {
            return fmt.Errorf("%w: team %s in semi but not in quarter",
                ErrBracketIncoherent, t)
        }
    }

    // teamsToFinal ⊂ teamsToSemiFinal
    sf := setOf(b.teamsToSemiFinal)
    for _, t := range b.teamsToFinal {
        if !sf[t] {
            return fmt.Errorf("%w: team %s in final but not in semi",
                ErrBracketIncoherent, t)
        }
    }
    return nil
}

func (v *BracketValidator) validateThirdPlace(b *BracketPrediction) error {
    // thirdPlaceWinner ∈ semifinalistas AND ∉ finalistas
    sf := setOf(b.teamsToSemiFinal)
    if !sf[b.thirdPlaceWinner] {
        return fmt.Errorf("%w: third place winner must be a semifinalist", ErrBracketIncoherent)
    }
    finalists := setOf(b.teamsToFinal)
    if finalists[b.thirdPlaceWinner] {
        return fmt.Errorf("%w: third place winner cannot be a finalist", ErrBracketIncoherent)
    }
    return nil
}

func (v *BracketValidator) validateChampion(b *BracketPrediction) error {
    // champion ∈ finalistas
    finalists := setOf(b.teamsToFinal)
    if !finalists[b.champion] {
        return fmt.Errorf("%w: champion must be a finalist", ErrBracketIncoherent)
    }
    return nil
}

func setOf(teams []team.ID) map[team.ID]bool {
    m := make(map[team.ID]bool, len(teams))
    for _, t := range teams {
        m[t] = true
    }
    return m
}

func isUnique(teams []team.ID) bool {
    seen := make(map[team.ID]bool, len(teams))
    for _, t := range teams {
        if seen[t] {
            return false
        }
        seen[t] = true
    }
    return true
}
```

## Errores

```go
// internal/domain/prediction/errors.go
package prediction

var (
    ErrBracketWindowClosed = errors.New("bracket prediction window is closed")
    ErrInvalidBracketSize  = errors.New("invalid number of teams")
    ErrBracketDuplicate    = errors.New("bracket has duplicate teams")
    ErrBracketIncoherent   = errors.New("bracket fails coherence check")
)
```

## Constructor con validación

```go
type NewBracketInput struct {
    UserID              user.ID
    PoolID              pool.ID
    TournamentID        tournament.ID
    TeamsToRoundOf32    []team.ID
    TeamsToRoundOf16    []team.ID
    TeamsToQuarterFinal []team.ID
    TeamsToSemiFinal    []team.ID
    TeamsToFinal        []team.ID
    ThirdPlaceWinner    team.ID
    Champion            team.ID
}

func NewBracketPrediction(
    in NewBracketInput,
    now time.Time,
    tournamentStartsAt time.Time,
    validator *BracketValidator,
) (*BracketPrediction, error) {
    if !now.Before(tournamentStartsAt) {
        return nil, ErrBracketWindowClosed
    }

    b := &BracketPrediction{
        id:                  ID(shared.NewID()),
        userID:              in.UserID,
        poolID:              in.PoolID,
        tournamentID:        in.TournamentID,
        teamsToRoundOf32:    in.TeamsToRoundOf32,
        teamsToRoundOf16:    in.TeamsToRoundOf16,
        teamsToQuarterFinal: in.TeamsToQuarterFinal,
        teamsToSemiFinal:    in.TeamsToSemiFinal,
        teamsToFinal:        in.TeamsToFinal,
        thirdPlaceWinner:    in.ThirdPlaceWinner,
        champion:            in.Champion,
        submittedAt:         now,
        updatedAt:           now,
    }

    if err := validator.Validate(b); err != nil {
        return nil, err
    }

    return b, nil
}
```

## Persistencia

En PostgreSQL, usar columnas array para los slices:

```sql
-- migrations/0006_bracket_predictions.up.sql
CREATE TABLE bracket_predictions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID NOT NULL REFERENCES users(id),
    pool_id                UUID NOT NULL REFERENCES pools(id),
    tournament_id          UUID NOT NULL REFERENCES tournaments(id),
    teams_to_round_of_32   UUID[] NOT NULL,
    teams_to_round_of_16   UUID[] NOT NULL,
    teams_to_quarter_final UUID[] NOT NULL,
    teams_to_semi_final    UUID[] NOT NULL,
    teams_to_final         UUID[] NOT NULL,
    third_place_winner     UUID NOT NULL REFERENCES teams(id),
    champion               UUID NOT NULL REFERENCES teams(id),
    submitted_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, pool_id, tournament_id),
    CHECK (array_length(teams_to_round_of_32, 1) = 32),
    CHECK (array_length(teams_to_round_of_16, 1) = 16),
    CHECK (array_length(teams_to_quarter_final, 1) = 8),
    CHECK (array_length(teams_to_semi_final, 1) = 4),
    CHECK (array_length(teams_to_final, 1) = 2)
);
```

Las `CHECK` constraints son una segunda barrera. La principal sigue siendo el validator en el dominio.

## Caso de uso: SubmitBracket

```go
// internal/application/commands/submit_bracket_prediction.go
package commands

type SubmitBracketPrediction struct {
    brackets    prediction.BracketRepository
    tournaments tournament.Repository
    validator   *prediction.BracketValidator
    clock       Clock
}

type SubmitBracketInput struct {
    UserID              string
    PoolID              string
    TournamentID        string
    TeamsToRoundOf32    []string
    TeamsToRoundOf16    []string
    TeamsToQuarterFinal []string
    TeamsToSemiFinal    []string
    TeamsToFinal        []string
    ThirdPlaceWinner    string
    Champion            string
}

func (uc *SubmitBracketPrediction) Execute(ctx context.Context, in SubmitBracketInput) error {
    t, err := uc.tournaments.FindByID(ctx, tournament.ID(in.TournamentID))
    if err != nil {
        return err
    }

    bracket, err := prediction.NewBracketPrediction(
        prediction.NewBracketInput{
            UserID:              user.ID(in.UserID),
            PoolID:              pool.ID(in.PoolID),
            TournamentID:        t.ID(),
            TeamsToRoundOf32:    toTeamIDs(in.TeamsToRoundOf32),
            // ... etc
        },
        uc.clock.Now(),
        t.StartsAt(),
        uc.validator,
    )
    if err != nil {
        return err
    }

    return uc.brackets.Upsert(ctx, bracket)
}
```

## UI Angular

El formulario de bracket es una **vista compleja**. Recomendado:

1. **Wizard de 6 pasos:**
   - Paso 1: seleccionar 32 equipos a octavos (de 48 disponibles)
   - Paso 2: seleccionar 16 de los 32
   - Paso 3: seleccionar 8 de los 16
   - Paso 4: seleccionar 4 de los 8
   - Paso 5: seleccionar 2 de los 4 (finalistas)
   - Paso 6: elegir campeón (de los 2) y tercer puesto (de los 2 no finalistas)

2. **Validación en cada paso:** no permitir avanzar hasta que el paso anterior esté completo.

3. **UI visual de "ir descartando":** los equipos eliminados se muestran tachados o difuminados.

4. **Botón "Save draft"** que envía a `/bracket/draft` (que NO valida coherencia completa, solo guarda) y "Submit final" que sí valida.

## Tests obligatorios

```go
func TestBracket_RejectsIfChampionNotInFinal(t *testing.T) {
    in := validBracketInput()
    in.Champion = team.ID("not-in-final")
    _, err := prediction.NewBracketPrediction(in, now, future, validator)
    assert.ErrorIs(t, err, prediction.ErrBracketIncoherent)
}

func TestBracket_RejectsAfterKickoff(t *testing.T) {
    in := validBracketInput()
    pastKickoff := time.Now().Add(-1 * time.Hour)
    _, err := prediction.NewBracketPrediction(in, time.Now(), pastKickoff, validator)
    assert.ErrorIs(t, err, prediction.ErrBracketWindowClosed)
}

func TestBracket_RejectsDuplicates(t *testing.T) {
    in := validBracketInput()
    in.TeamsToRoundOf32[0] = in.TeamsToRoundOf32[1]
    _, err := prediction.NewBracketPrediction(in, now, future, validator)
    assert.ErrorIs(t, err, prediction.ErrBracketDuplicate)
}

func TestBracket_RejectsThirdPlaceAsFinalist(t *testing.T) {
    in := validBracketInput()
    in.ThirdPlaceWinner = in.TeamsToFinal[0]
    _, err := prediction.NewBracketPrediction(in, now, future, validator)
    assert.ErrorIs(t, err, prediction.ErrBracketIncoherent)
}
```
