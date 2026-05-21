# Modelo de dominio

Este documento describe el modelo de dominio del sistema de quinielas. Es la fuente de verdad para entender qué entidades existen, qué reglas las gobiernan y cómo se relacionan.

## Bounded Contexts

Aunque es un sistema relativamente pequeño, conviene identificar contextos:

1. **Identidad y acceso** — Usuarios, autenticación, invitaciones
2. **Quiniela** — Grupos (Pools), membresías, configuración
3. **Torneo** — Estructura del Mundial: equipos, grupos del torneo, partidos, fases
4. **Pronóstico** — Pronósticos de partido y de bracket
5. **Puntuación** — Reglas, cálculo, ranking

## Glosario crítico

| Término | Significado |
|---------|-------------|
| **Pool** | Grupo privado de quiniela. Tiene N integrantes que pronostican el mismo Mundial. |
| **Group (del Mundial)** | Grupo del torneo (Grupo A, B, ..., L). 12 grupos de 4 equipos. |
| **Stage** | Fase del torneo: `group`, `round_of_32`, `round_of_16`, `quarter_final`, `semi_final`, `third_place`, `final`. |
| **Match** | Partido individual del torneo. |
| **Match Prediction** | Pronóstico de marcador (goles local + goles visitante) para un partido específico. |
| **Bracket Prediction** | Pronóstico del avance completo del torneo: qué equipos pasan a cada fase + tercer puesto + campeón. |

**Nota terminológica:** uso "Pool" para grupos privados y "Group" para grupos del Mundial para evitar el choque semántico. En el código y la UI mantenerlo así.

---

## Agregados

### 1. User (Identidad y acceso)

**Aggregate Root:** `User`

```
User
├── id: UserID
├── email: Email (VO)
├── passwordHash: string
├── displayName: string
├── createdAt: time.Time
└── verifiedAt: *time.Time
```

**Invariantes:**
- Email único en el sistema.
- Email debe ser válido (formato RFC 5322 básico).
- `displayName` entre 2 y 50 caracteres.

---

### 2. Pool (Quiniela)

**Aggregate Root:** `Pool`

```
Pool
├── id: PoolID
├── name: string
├── description: string
├── creatorID: UserID
├── tournamentID: TournamentID
├── settings: PoolSettings (VO)
├── createdAt: time.Time
└── members: []PoolMember

PoolMember
├── userID: UserID
├── role: MemberRole (creator | admin | member)
├── joinedAt: time.Time
└── invitedBy: *UserID

Invitation
├── id: InvitationID
├── poolID: PoolID
├── email: Email
├── token: string
├── invitedBy: UserID
├── expiresAt: time.Time
└── acceptedAt: *time.Time

PoolSettings (Value Object)
├── matchPredictionCutoffMinutes: int  (default: 0 = al kickoff)
├── extraTimeRule: ExtraTimeRule       (regular | final_official)
└── showOtherPredictions: bool          (default: true tras cierre)
```

**Invariantes del agregado:**
- Un Pool tiene exactamente un creador (`creator` role).
- Los `admin` son designados por el creador.
- Los miembros se unen vía `Invitation` aceptada con token válido.
- `name` entre 3 y 80 caracteres.
- Una `Invitation` expira a las 7 días por default.
- Un email no puede tener dos invitaciones activas en el mismo Pool.

**Reglas de negocio:**
- Solo el `creator` puede transferir su rol o eliminar el Pool.
- `creator` y `admin` pueden invitar nuevos miembros.
- Cualquier miembro puede salirse, salvo el `creator` (debe transferir primero).

---

### 3. Tournament (Torneo)

**Aggregate Root:** `Tournament`

```
Tournament
├── id: TournamentID
├── name: string                  ("FIFA World Cup 2026")
├── startsAt: time.Time           (kickoff del partido inaugural)
├── endsAt: time.Time             (final)
├── teams: []Team
├── groups: []Group               (12 grupos del Mundial)
└── stages: []StageDefinition

Team
├── id: TeamID
├── code: string                  ("ARG", "BRA", ...)
├── name: string
├── flagURL: string
└── confederation: string

Group (del Mundial)
├── id: GroupID                   ("A".."L")
├── tournamentID: TournamentID
└── teams: []TeamID               (4 equipos)

StageDefinition
├── stage: Stage
├── pointsPerCorrectTeam: int     (3, 4, 5, 10, 15, 20)
└── description: string
```

**Invariantes:**
- 48 teams en el Mundial 2026.
- 12 groups con exactamente 4 teams cada uno.
- `startsAt` < `endsAt`.
- Las `StageDefinition` siguen las reglas del usuario: 3 pts (octavos), 4 (cuartos), 5 (semi), 10 (final), 15 (tercer puesto), 20 (campeón).

**Nota sobre Mundial 2026:** 48 equipos, 12 grupos de 4, los 2 primeros de cada grupo (24) + los 8 mejores terceros = 32 equipos a octavos. Esto es **nuevo formato**; tu sistema debe modelarlo así, no como Mundiales anteriores de 32 equipos.

---

### 4. Match (Partido)

**Aggregate Root:** `Match`

```
Match
├── id: MatchID
├── tournamentID: TournamentID
├── stage: Stage
├── groupID: *GroupID             (solo en fase de grupos)
├── homeTeamID: TeamID
├── awayTeamID: TeamID
├── kickoffAt: time.Time          (con timezone)
├── venue: string
├── status: MatchStatus           (scheduled | in_progress | finished | cancelled)
└── result: *MatchResult          (nil hasta que termine)

MatchResult (Value Object)
├── homeGoals: int
├── awayGoals: int
├── homeGoalsAfterET: *int        (post prórroga si aplica)
├── awayGoalsAfterET: *int
├── homeGoalsAfterPenalties: *int (post penales si aplica)
├── awayGoalsAfterPenalties: *int
├── finalizedAt: time.Time
└── source: ResultSource          (api_footballdata | api_balldontlie | manual)
```

**Invariantes:**
- `homeTeamID != awayTeamID`.
- En fase de grupos, ambos equipos pertenecen al mismo `groupID`.
- Una vez `status = finished` con `result != nil`, el resultado es inmutable salvo por un admin que lo corrija explícitamente (con auditoría).

**Regla post-penales:** según decisión del usuario, en eliminación directa el **resultado oficial post-penales** es el que cuenta para acertar "ganador". Implementación:

```go
func (r MatchResult) OfficialWinner(stage Stage) Winner {
    if stage == StageGroup {
        // Solo cuenta tiempo regular (no hay prórroga ni penales en grupos)
        return winnerFromScore(r.homeGoals, r.awayGoals)
    }
    // Eliminación directa: usar resultado final
    if r.homeGoalsAfterPenalties != nil {
        return winnerFromScore(*r.homeGoalsAfterPenalties, *r.awayGoalsAfterPenalties)
    }
    if r.homeGoalsAfterET != nil {
        return winnerFromScore(*r.homeGoalsAfterET, *r.awayGoalsAfterET)
    }
    return winnerFromScore(r.homeGoals, r.awayGoals)
}
```

**Pero los goles que cuentan para "acertar goles" son los del tiempo regular** (porque ahí es donde el usuario pronostica). Esto es importante: para el cálculo de 1+1 puntos por acertar goles, usar `homeGoals`/`awayGoals` (tiempo regular). Para los 3 puntos de "acertar ganador", usar `OfficialWinner()`.

---

### 5. MatchPrediction (Pronóstico de partido)

**Aggregate Root:** `MatchPrediction`

```
MatchPrediction
├── id: PredictionID
├── userID: UserID
├── poolID: PoolID
├── matchID: MatchID
├── homeGoals: int                (≥ 0)
├── awayGoals: int                (≥ 0)
├── submittedAt: time.Time
└── updatedAt: time.Time
```

**Invariantes:**
- `homeGoals ≥ 0` y `awayGoals ≥ 0`.
- `homeGoals ≤ 30` y `awayGoals ≤ 30` (sanity check; nadie pronostica goleadas mayores).
- Único por `(userID, poolID, matchID)`.
- **No se puede crear ni modificar** si `now() >= match.kickoffAt - pool.settings.matchPredictionCutoffMinutes`.

**Operación clave:**
```go
type MatchPrediction struct { ... }

func (p *MatchPrediction) Update(homeGoals, awayGoals int, now time.Time, match *Match) error {
    if now.After(match.KickoffAt()) || now.Equal(match.KickoffAt()) {
        return ErrPredictionWindowClosed
    }
    if homeGoals < 0 || awayGoals < 0 {
        return ErrInvalidScore
    }
    p.homeGoals = homeGoals
    p.awayGoals = awayGoals
    p.updatedAt = now
    return nil
}
```

---

### 6. BracketPrediction (Pronóstico de bracket)

**Aggregate Root:** `BracketPrediction`

```
BracketPrediction
├── id: BracketPredictionID
├── userID: UserID
├── poolID: PoolID
├── tournamentID: TournamentID
├── teamsToRoundOf32: [32]TeamID  (equipos que pasan de fase de grupos)
├── teamsToRoundOf16: [16]TeamID  (equipos que pasan a octavos→cuartos)
├── teamsToQuarterFinal: [8]TeamID
├── teamsToSemiFinal: [4]TeamID
├── teamsToFinal: [2]TeamID
├── thirdPlaceWinner: TeamID
├── champion: TeamID
├── submittedAt: time.Time
└── updatedAt: time.Time
```

**Invariante clave de coherencia (jerarquía):**
```
champion ∈ teamsToFinal
teamsToFinal ⊂ teamsToSemiFinal
teamsToSemiFinal ⊂ teamsToQuarterFinal
teamsToQuarterFinal ⊂ teamsToRoundOf16
teamsToRoundOf16 ⊂ teamsToRoundOf32
thirdPlaceWinner ∈ teamsToSemiFinal AND thirdPlaceWinner ∉ teamsToFinal
```

El servicio de dominio `BracketCoherenceValidator` valida esta jerarquía completa.

**Invariante de cierre:**
- **No se puede crear ni modificar** si `now() >= tournament.startsAt`.

**Decisión consciente:** se mantiene una sola fila por `(userID, poolID)`. El bracket es un objeto único editable hasta el cierre, no un histórico.

---

### 7. Scoring (Puntuación)

**Tabla `score_entries`** (cada cómputo de puntos es una fila auditable):

```
ScoreEntry
├── id: ScoreEntryID
├── userID: UserID
├── poolID: PoolID
├── sourceType: ScoreSourceType    (match | bracket_stage | bracket_third | bracket_champion)
├── sourceRef: string              (match_id, stage_name, ...)
├── points: int
├── computedAt: time.Time
└── version: int                   (para idempotencia/recálculo)
```

**Ver `SCORING-RULES.md` para detalle de cálculo.**

**Ranking** se obtiene como query agregada:
```sql
SELECT user_id, SUM(points) AS total
FROM score_entries
WHERE pool_id = $1
GROUP BY user_id
ORDER BY total DESC;
```

---

## Eventos de dominio

Eventos publicados al `EventBus` interno:

| Evento | Cuándo | Suscriptores |
|--------|--------|--------------|
| `MatchResultFinalized` | Admin/cron carga resultado final | `ScoringEngine` → calcula puntos del partido |
| `GroupStageCompleted` | Se finaliza el último partido de grupos | `ScoringEngine` → calcula puntos de octavos del bracket |
| `RoundOfThirtyTwoCompleted` | Termina ronda de 32 | `ScoringEngine` → puntos de cuartos |
| `RoundOfSixteenCompleted` | Termina ronda de 16 | `ScoringEngine` → puntos de semis |
| `QuarterFinalsCompleted` | Terminan cuartos | `ScoringEngine` → puntos de finalistas |
| `SemiFinalsCompleted` | Terminan semis | `ScoringEngine` → puntos de campeón + tercer puesto |
| `InvitationAccepted` | Usuario acepta invitación | (futuro: notificaciones) |

## Identificadores

Usar **UUID v7** (sortable timestamp-based) para todos los IDs. Razones:
- Únicos sin coordinación entre nodos
- Sortables por tiempo de creación (útil para paginación)
- Tipo `string` en sqlc, `uuid` en PostgreSQL

```go
type UserID string
type PoolID string
type MatchID string
// ... etc
```

## Resumen visual de agregados

```
User ────────┬───→ Pool ←─── PoolMember
             │      │
             │      └─── Invitation
             │
             ├──→ MatchPrediction ───→ Match ←─── Tournament
             │                              └─── Group (del Mundial)
             │                              └─── Team
             │
             ├──→ BracketPrediction ──→ Tournament
             │                            └─── Team (referenciado en cada fase)
             │
             └──→ ScoreEntry (Pool, source)
```
