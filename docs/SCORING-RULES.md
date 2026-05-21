# Reglas de puntaje (Scoring Rules)

Especificación **formal y exhaustiva** del sistema de puntaje. Este documento es el contrato del sistema: cualquier cambio aquí debe reflejarse en código y tests.

---

## 1. Pronóstico de partido (Match Prediction)

Por cada partido del torneo, el usuario pronostica los goles de cada equipo. El cálculo de puntos se hace cuando el partido está finalizado.

### 1.1 Acertar ganador o empate — 3 puntos

Sea:
- `pred.home_goals`, `pred.away_goals` — pronóstico del usuario
- `result.home_goals_official`, `result.away_goals_official` — resultado **oficial** (post-penales en eliminación directa, tiempo regular en fase de grupos)

El "ganador pronosticado" se define por la **dirección del resultado**:

```
pred_winner = HOME    if pred.home_goals > pred.away_goals
pred_winner = AWAY    if pred.home_goals < pred.away_goals
pred_winner = DRAW    if pred.home_goals == pred.away_goals

actual_winner = HOME  if result.home_goals_official > result.away_goals_official
actual_winner = AWAY  if result.home_goals_official < result.away_goals_official
actual_winner = DRAW  if result.home_goals_official == result.away_goals_official
```

**Otorgar 3 puntos si y solo si `pred_winner == actual_winner`.**

**Caso especial — eliminación directa:** un partido de octavos puede empatar 1-1 en tiempo regular y definirse por penales. Si el usuario pronosticó "empate 1-1":
- El "ganador" oficial es el que ganó por penales (HOME o AWAY).
- El usuario predijo DRAW.
- **No otorga los 3 puntos** porque `DRAW != HOME/AWAY`.
- Pero **sí otorga puntos por acertar goles** del tiempo regular (sección 1.2).

### 1.2 Acertar goles de un equipo — 1 punto por equipo

Se evalúa contra el **resultado del tiempo regular** (no post-penales), porque el usuario pronostica un marcador "normal" del partido.

```
SI pred.home_goals == result.home_goals (tiempo regular): +1 punto
SI pred.away_goals == result.away_goals (tiempo regular): +1 punto
```

### 1.3 Máximo por partido: 5 puntos

| Caso | Puntos |
|------|--------|
| Acierta ganador + ambos goles | 3 + 1 + 1 = **5** |
| Acierta ganador + un solo gol | 3 + 1 = **4** |
| Acierta ganador, ningún gol | **3** |
| No acierta ganador pero acierta ambos goles | 0 + 1 + 1 = **2** |
| No acierta ganador, un solo gol | 0 + 1 = **1** |
| No acierta nada | **0** |

> **Nota:** acertar ambos goles exactos sin acertar ganador es imposible cuando hay diferencia entre los goles, así que el caso "2 puntos" solo ocurre con empate. Ejemplo: predices 1-1 (DRAW) y el partido fue 1-1 en regular pero ganó USA por penales en octavos → empatas ambos goles del tiempo regular pero no aciertas ganador oficial.

### 1.4 Pseudocódigo de referencia

```go
func ComputeMatchPoints(pred MatchPrediction, match Match, result MatchResult) int {
    points := 0

    // Goles del tiempo regular (los que el usuario pronosticó)
    if pred.HomeGoals == result.HomeGoals {
        points += 1
    }
    if pred.AwayGoals == result.AwayGoals {
        points += 1
    }

    // Ganador oficial (post-penales si aplica)
    predWinner := winnerOf(pred.HomeGoals, pred.AwayGoals)
    actualWinner := result.OfficialWinner(match.Stage)
    if predWinner == actualWinner {
        points += 3
    }

    return points
}

func winnerOf(home, away int) Winner {
    switch {
    case home > away: return WinnerHome
    case home < away: return WinnerAway
    default: return WinnerDraw
    }
}
```

---

## 2. Pronóstico de bracket (Bracket Prediction)

El usuario pronostica, antes del kickoff inaugural, qué equipos avanzan a cada fase del torneo, más el campeón y el ganador del tercer puesto.

### 2.1 Puntos por fase alcanzada

**Por cada equipo del pronóstico que efectivamente alcanza esa fase, se otorgan puntos.**

| Fase del bracket | Equipos pronosticados | Puntos por equipo acertado |
|---|---|---|
| Pasa a octavos (Round of 32) | 32 | **3 pts** c/u |
| Pasa a cuartos (Round of 16) | 16 | **4 pts** c/u |
| Pasa a semifinal | 8 | **5 pts** c/u |
| Llega a final | 4 | **10 pts** c/u |

> **Nota terminológica importante:** en Mundial 2026, "octavos" es la ronda de 32 equipos (Round of 32), porque hay 48 equipos divididos en 12 grupos. La "ronda de 16" (Round of 16) corresponde a la fase eliminatoria de 16 equipos previa a cuartos. Mantener esta nomenclatura consistente en código y UI.

### 2.2 Puntos por aciertos únicos

| Acierto | Puntos |
|---|---|
| Ganador del tercer puesto | **15 pts** |
| Campeón | **20 pts** |

### 2.3 Pseudocódigo de referencia

```go
func ComputeBracketStagePoints(
    pred BracketPrediction,
    actualTeamsAtStage map[Stage][]TeamID,
) []ScoreEntry {
    var entries []ScoreEntry

    // Por cada fase
    stageRules := []struct {
        stage    Stage
        predicted []TeamID
        points   int
    }{
        {StageRoundOf32, pred.TeamsToRoundOf32, 3},
        {StageRoundOf16, pred.TeamsToRoundOf16, 4},
        {StageSemiFinal, pred.TeamsToSemiFinal, 5},
        {StageFinal,     pred.TeamsToFinal,     10},
    }

    for _, sr := range stageRules {
        actualSet := setOf(actualTeamsAtStage[sr.stage])
        for _, team := range sr.predicted {
            if actualSet.Contains(team) {
                entries = append(entries, ScoreEntry{
                    SourceType: ScoreSourceBracketStage,
                    SourceRef:  fmt.Sprintf("%s:%s", sr.stage, team),
                    Points:     sr.points,
                })
            }
        }
    }

    // Tercer puesto
    if actualThirdPlace == pred.ThirdPlaceWinner {
        entries = append(entries, ScoreEntry{
            SourceType: ScoreSourceBracketThirdPlace,
            SourceRef:  "third_place",
            Points:     15,
        })
    }

    // Campeón
    if actualChampion == pred.Champion {
        entries = append(entries, ScoreEntry{
            SourceType: ScoreSourceBracketChampion,
            SourceRef:  "champion",
            Points:     20,
        })
    }

    return entries
}
```

### 2.4 Cuándo se computan los puntos de bracket

El motor escucha eventos de fase completada y va computando puntos en cascada:

| Evento (cuándo) | Computa puntos de |
|---|---|
| `GroupStageCompleted` (últimas jornadas de grupos terminadas) | Octavos (Round of 32) |
| `RoundOfThirtyTwoCompleted` | Ronda de 16 |
| `RoundOfSixteenCompleted` | Cuartos (semis predichos) |
| `QuarterFinalsCompleted` | Finalistas |
| `SemiFinalsCompleted` | Tercer puesto + Campeón (esperar partidos finales) |
| `ThirdPlaceMatchFinalized` | Confirma tercer puesto |
| `FinalMatchFinalized` | Confirma campeón |

> **Idempotencia:** cada `ScoreEntry` tiene `(user_id, pool_id, source_type, source_ref)` como llave única. Recalcular es seguro porque hace UPSERT, no INSERT puro.

---

## 3. Ranking del grupo

El ranking de un Pool es la suma de todos los `ScoreEntry` de sus miembros, ordenado descendente.

```sql
SELECT
    u.id AS user_id,
    u.display_name,
    COALESCE(SUM(se.points), 0) AS total_points,
    COUNT(se.id) FILTER (WHERE se.source_type = 'match') AS match_hits,
    COUNT(se.id) FILTER (WHERE se.source_type LIKE 'bracket%') AS bracket_hits
FROM users u
JOIN pool_members pm ON pm.user_id = u.id
LEFT JOIN score_entries se ON se.user_id = u.id AND se.pool_id = pm.pool_id
WHERE pm.pool_id = $1
GROUP BY u.id, u.display_name
ORDER BY total_points DESC, match_hits DESC, u.display_name ASC;
```

**Desempate:**
1. Mayor cantidad de pronósticos de partido acertados (al menos 3 pts).
2. Si persiste, orden alfabético por nombre (estable, sin sorteo).

> **El ganador de la quiniela** es el primero del ranking al cerrarse el torneo (después del partido final).

---

## 4. Puntajes máximos teóricos

Útil como referencia y para validar tests:

- **Pronósticos de partido (104 partidos × 5 pts máx):** 520 pts
- **Pronóstico de bracket:**
  - Octavos: 32 × 3 = 96
  - Ronda de 16: 16 × 4 = 64
  - Semifinal: 8 × 5 = 40
  - Final: 4 × 10 = 40
  - Tercer puesto: 15
  - Campeón: 20
  - **Total bracket: 275 pts**
- **Máximo teórico total: 795 pts**

Un usuario "promedio" probablemente esté entre 80-200 pts. Esto es referencial; cada quiniela varía.

---

## 5. Casos edge documentados

### 5.1 Partido cancelado
Si un partido se cancela (terremoto, walkover sin disputa, etc.), no se otorgan puntos por él. Estado del match queda como `cancelled`.

### 5.2 Partido reprogramado
Si un partido se mueve de fecha **antes del kickoff original**, el `kickoffAt` se actualiza y los pronósticos siguen siendo modificables hasta el nuevo kickoff.

### 5.3 Resultado corregido a posteriori
Si un admin corrige un resultado ya cargado (error en data), el motor recalcula los puntos del partido. Los `ScoreEntry` con la nueva llave reemplazan a los anteriores (UPSERT por `(user, pool, source_type, source_ref)`).

### 5.4 Empate en eliminación directa con penales
- **Goles del tiempo regular** (homeGoals/awayGoals): los que el usuario predijo. Cuentan para el "+1 por gol".
- **Ganador oficial:** post-penales. Cuenta para los "+3 por ganador".
- Un usuario que predijo "1-1" en un partido que terminó 1-1 (luego ganó USA por penales) obtiene 1+1 = 2 puntos, NO 3.

### 5.5 Equipo descalificado
Si un equipo es descalificado del torneo, todos los pronósticos de bracket que lo incluyan en cualquier fase pierden esos puntos (porque ese equipo nunca "alcanza" la fase). Esto se maneja automáticamente porque el motor mira los equipos que **efectivamente alcanzaron** la fase.

---

## 6. Tests obligatorios

Los siguientes tests **deben existir** y pasar antes de declarar el motor de scoring funcional:

- [ ] Acertar marcador exacto en fase de grupos = 5 pts
- [ ] Acertar solo ganador = 3 pts
- [ ] Predicción 0-0 vs resultado 0-0 = 5 pts
- [ ] Predicción 1-2 vs resultado 1-2 (empate por reglas raras... NO se da, pero validar 1-2 vs 1-2) = 5 pts
- [ ] Predicción 1-1 vs resultado 1-1 en grupos = 5 pts (empate exacto)
- [ ] Predicción 1-1 vs resultado 1-1 en octavos donde ganó local por penales = 2 pts
- [ ] Predicción 2-1 vs resultado 1-2 = 0 pts (perdedor opuesto)
- [ ] Recalcular puntos dos veces no duplica entries
- [ ] Cargar resultado de partido inexistente devuelve error
- [ ] Bracket: 32 equipos predichos correctos = 96 pts en evento `GroupStageCompleted`
- [ ] Bracket: predecir campeón equipo X cuando X no llega a final = 0 pts
- [ ] Bracket: predecir tercer puesto correctamente pero perdió la semifinal contra el campeón = 15 pts
