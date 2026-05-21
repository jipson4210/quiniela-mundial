# Integración con APIs externas

Estrategia híbrida para cargar el fixture del Mundial y mantener resultados actualizados sin depender de una sola fuente.

## Principio rector

**Ninguna API es 100% confiable.** El sistema debe poder operar manualmente si las APIs fallan en el momento crítico (durante un partido).

## Las tres fuentes

### 1. openfootball/worldcup.json (Bootstrap inicial)

**URL:** `https://raw.githubusercontent.com/openfootball/worldcup.json/master/2026/worldcup.json`

**Uso:** carga única al hacer seed inicial. Contiene los 104 partidos, 48 equipos, 12 grupos y horarios.

**Ventajas:**
- Sin API key
- Sin rate limits
- Dominio público
- Estructura JSON limpia

**Desventajas:**
- Solo datos estáticos (no resultados en vivo)
- Actualizaciones dependen de commits del mantenedor

**Cuando usar:** `cmd/api seed --tournament=worldcup-2026`

### 2. football-data.org (Sync de resultados, primario)

**URL base:** `https://api.football-data.org/v4/`

**Auth:** API key gratuita (`X-Auth-Token` header)

**Rate limit free tier:** 10 requests/minuto

**Uso:** cron job durante días de partido pidiendo solo partidos del día actual.

**Endpoint clave:** `GET /v4/competitions/WC/matches?dateFrom=YYYY-MM-DD&dateTo=YYYY-MM-DD`

**Ventajas:**
- Confiable y rápido
- Buena documentación
- Cubre Mundial sin restricciones extra en free tier

**Estrategia de caching:** cada respuesta se cachea 60 segundos para no quemar rate limit.

### 3. balldontlie.io (Fallback)

**URL base:** `https://api.balldontlie.io/v1/`

**Auth:** API key gratuita

**Uso:** si football-data.org devuelve error 5xx o timeout en 3 intentos consecutivos.

**Ventajas:**
- Cobertura del Mundial 2026 completa
- Stats adicionales (xG, momentum)

### 4. Fallback manual (último recurso)

**Endpoint admin:** `POST /api/v1/admin/matches/:id/result`

Permite al administrador (rol `creator` o `admin` de Pool con permiso global) cargar resultado a mano. Esto **debe existir desde el día 1** — es lo que te salva si Brasil-Argentina termina y las APIs están saturadas.

---

## Patrón de implementación: Adapter

Todas las fuentes se exponen detrás de una **única interfaz de dominio**, definida en `internal/domain/match/`:

```go
package match

type ResultProvider interface {
    // FetchResults pide los resultados de partidos en un rango de fechas.
    // Devuelve solo partidos finalizados.
    FetchResults(ctx context.Context, from, to time.Time) ([]ExternalResult, error)
}

type ExternalResult struct {
    ExternalMatchID         string         // ID en la fuente externa
    HomeTeamCode            string         // "ARG", "BRA"...
    AwayTeamCode            string
    KickoffAt               time.Time
    Status                  string         // "FINISHED", "IN_PLAY"...
    HomeGoals               *int
    AwayGoals               *int
    HomeGoalsAfterET        *int
    AwayGoalsAfterET        *int
    HomeGoalsAfterPenalties *int
    AwayGoalsAfterPenalties *int
    Source                  string
    FetchedAt               time.Time
}
```

Cada adapter en `internal/infrastructure/external/` implementa esta interfaz:

```
internal/infrastructure/external/
├── openfootball/
│   ├── client.go
│   └── adapter.go        // Implementa TournamentSeedProvider
├── footballdata/
│   ├── client.go
│   ├── dto.go            // DTOs de la respuesta JSON
│   └── adapter.go        // Implementa ResultProvider
└── balldontlie/
    ├── client.go
    ├── dto.go
    └── adapter.go        // Implementa ResultProvider
```

---

## Composición con Chain of Responsibility

El sistema usa un **provider con fallback en cadena**:

```go
type ChainedResultProvider struct {
    primary  ResultProvider
    fallback ResultProvider
    logger   Logger
}

func (c *ChainedResultProvider) FetchResults(ctx context.Context, from, to time.Time) ([]ExternalResult, error) {
    results, err := c.primary.FetchResults(ctx, from, to)
    if err == nil {
        return results, nil
    }
    c.logger.Warn("primary provider failed, falling back", "err", err)
    return c.fallback.FetchResults(ctx, from, to)
}
```

En `wire.go` se compone:
```go
chained := external.NewChainedResultProvider(
    footballdata.NewAdapter(cfg.FootballDataKey),
    balldontlie.NewAdapter(cfg.BallDontLieKey),
    logger,
)
```

---

## Job de sincronización

Ubicación: `internal/interfaces/jobs/result_sync_job.go`

**Frecuencia:** cada 5 minutos durante días con partidos. Fuera de esos días, una vez al día (por si hay actualizaciones tardías).

**Estrategia:**
1. Determinar ventana: `[ahora - 6h, ahora + 3h]` (cubre partidos que pueden estar en juego o recién terminados).
2. Pedir resultados al provider con la ventana.
3. Por cada `ExternalResult` con status `FINISHED`:
   - Mapear `HomeTeamCode` y `AwayTeamCode` a `TeamID` interno.
   - Encontrar el `Match` correspondiente por `(tournamentID, homeTeam, awayTeam, kickoffAt)`.
   - Si el match no tiene `result` aún, crearlo.
   - Si ya tiene `result`, comparar y solo actualizar si difiere (con auditoría).
4. Publicar evento `MatchResultFinalized` para cada match recién finalizado.

**Idempotencia:** el job puede correr 100 veces seguidas sin duplicar nada.

**Manejo de errores:**
- Error de red transient → retry exponencial (3 intentos: 1s, 2s, 4s)
- Error 4xx (auth, mal request) → log + alert + no retry
- Resultado con goles diferentes a los almacenados → log warning + sobrescribir solo si la fuente es de mayor confianza que la actual

---

## Mapeo de equipos (problema clásico)

Cada fuente usa códigos distintos:
- openfootball: códigos FIFA de 3 letras (`ARG`, `BRA`, `ECU`)
- football-data.org: `name` completo en inglés (`Argentina`, `Brazil`)
- balldontlie: IDs propios

**Solución:** tabla `team_external_ids` que mapea cada `TeamID` interno a su identificador en cada fuente.

```sql
CREATE TABLE team_external_ids (
    team_id     UUID NOT NULL REFERENCES teams(id),
    source      VARCHAR(50) NOT NULL,  -- 'openfootball', 'footballdata', 'balldontlie'
    external_id VARCHAR(100) NOT NULL,
    PRIMARY KEY (team_id, source)
);
```

Esta tabla se llena durante el seed inicial (Fase 1 del roadmap) cruzando los datos de openfootball con los nombres en otras fuentes.

---

## Resiliencia

### Circuit breaker

Para cada adapter externo, usar un circuit breaker (paquete `sony/gobreaker`):
- Si falla 5 veces seguidas, abre el circuito por 60 segundos.
- En estado abierto, devuelve error inmediato sin pegarle a la API.
- Tras 60s pasa a half-open, prueba con 1 request.

Esto evita martillar una API caída y ahorra rate limit.

### Timeout

Cada request HTTP tiene timeout de **10 segundos**. Si la API tarda más, se considera fallida.

### Métricas

Loguear en formato estructurado:
- `provider`, `endpoint`, `duration_ms`, `status_code`, `error`
- Conteo de partidos sincronizados por corrida

(Opcional para Fase 1; útil cuando ya está en producción.)

---

## Configuración

Variables de entorno:

```bash
# football-data.org
FOOTBALLDATA_API_KEY=xxx
FOOTBALLDATA_BASE_URL=https://api.football-data.org/v4

# balldontlie
BALLDONTLIE_API_KEY=xxx
BALLDONTLIE_BASE_URL=https://api.balldontlie.io/v1

# Sync
SYNC_INTERVAL_DURING_MATCHDAY=5m
SYNC_INTERVAL_OFFDAY=24h
SYNC_LOOKBACK_HOURS=6
SYNC_LOOKAHEAD_HOURS=3
```

---

## Checklist de implementación

- [ ] Adapter `openfootball` para seed inicial
- [ ] Adapter `footballdata` para resultados
- [ ] Adapter `balldontlie` como fallback
- [ ] `ChainedResultProvider` con fallback
- [ ] Tabla `team_external_ids` y seed con mapeos
- [ ] Job de sincronización idempotente
- [ ] Circuit breaker en cada adapter
- [ ] Endpoint admin manual `POST /admin/matches/:id/result`
- [ ] Tests con mocks HTTP (`httptest`)
- [ ] Documentar API keys en `.env.example`
