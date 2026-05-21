---
name: external-api-adapter
description: How to implement adapters for external football APIs (openfootball, football-data.org, balldontlie) following the Adapter pattern. Use this skill when integrating any external data source, when building or modifying anything in /internal/infrastructure/external/, when implementing the ResultProvider interface, when writing the fixture sync job, or when adding team-code mappings. Apply this for any HTTP client code that talks to a third-party API.
---

# External API Adapters

Cómo integrar APIs externas (openfootball, football-data.org, balldontlie) siguiendo el **Adapter Pattern**.

**Ver `docs/API-INTEGRATION.md` para la estrategia completa.**

## Principio

Cada API externa habla su propio idioma. El dominio habla **uno solo**. Los adapters traducen.

```
domain.ResultProvider (interfaz)
        ↑           ↑           ↑
    footballdata  balldontlie  openfootball
       adapter     adapter      adapter
```

## Estructura de carpetas

```
internal/infrastructure/external/
├── openfootball/
│   ├── client.go       # HTTP cliente
│   ├── dto.go          # DTOs del JSON
│   ├── adapter.go      # Implementa TournamentSeedProvider
│   └── adapter_test.go
├── footballdata/
│   ├── client.go
│   ├── dto.go
│   ├── adapter.go      # Implementa ResultProvider
│   └── adapter_test.go
├── balldontlie/
│   ├── client.go
│   ├── dto.go
│   └── adapter.go
└── chained/
    └── result_provider.go  # Combinador con fallback
```

## Plantilla: Cliente HTTP

```go
// internal/infrastructure/external/footballdata/client.go
package footballdata

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        baseURL: baseURL,
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *Client) get(ctx context.Context, path string, out any) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
    if err != nil {
        return err
    }
    req.Header.Set("X-Auth-Token", c.apiKey)
    req.Header.Set("Accept", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("http request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
    }

    return json.NewDecoder(resp.Body).Decode(out)
}
```

## Plantilla: DTOs

```go
// internal/infrastructure/external/footballdata/dto.go
package footballdata

import "time"

type matchesResponse struct {
    Matches []matchDTO `json:"matches"`
}

type matchDTO struct {
    ID       int       `json:"id"`
    UtcDate  time.Time `json:"utcDate"`
    Status   string    `json:"status"`  // SCHEDULED, IN_PLAY, FINISHED, ...
    HomeTeam teamDTO   `json:"homeTeam"`
    AwayTeam teamDTO   `json:"awayTeam"`
    Score    scoreDTO  `json:"score"`
    Stage    string    `json:"stage"`
}

type teamDTO struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    TLA  string `json:"tla"`
}

type scoreDTO struct {
    Winner   string         `json:"winner"`
    Duration string         `json:"duration"`  // REGULAR, EXTRA_TIME, PENALTY_SHOOTOUT
    FullTime scoreRecord   `json:"fullTime"`
    HalfTime scoreRecord   `json:"halfTime"`
    Regular  *scoreRecord  `json:"regularTime,omitempty"`
    ExtraTime *scoreRecord `json:"extraTime,omitempty"`
    Penalties *scoreRecord `json:"penalties,omitempty"`
}

type scoreRecord struct {
    Home *int `json:"home"`
    Away *int `json:"away"`
}
```

> **Nota:** los DTOs son privados (`lowerCase`). Nunca exportarlos fuera del paquete. La conversión a tipos del dominio se hace en el adapter.

## Plantilla: Adapter

```go
// internal/infrastructure/external/footballdata/adapter.go
package footballdata

import (
    "context"
    "time"

    "github.com/<repo>/internal/domain/match"
)

type Adapter struct {
    client *Client
}

func NewAdapter(client *Client) *Adapter {
    return &Adapter{client: client}
}

// FetchResults implementa match.ResultProvider.
func (a *Adapter) FetchResults(
    ctx context.Context,
    from, to time.Time,
) ([]match.ExternalResult, error) {
    path := fmt.Sprintf("/v4/competitions/WC/matches?dateFrom=%s&dateTo=%s",
        from.Format("2006-01-02"),
        to.Format("2006-01-02"),
    )

    var resp matchesResponse
    if err := a.client.get(ctx, path, &resp); err != nil {
        return nil, fmt.Errorf("fetch results: %w", err)
    }

    results := make([]match.ExternalResult, 0, len(resp.Matches))
    for _, m := range resp.Matches {
        results = append(results, toExternalResult(m))
    }
    return results, nil
}

func toExternalResult(m matchDTO) match.ExternalResult {
    r := match.ExternalResult{
        ExternalMatchID: strconv.Itoa(m.ID),
        HomeTeamCode:    m.HomeTeam.TLA,
        AwayTeamCode:    m.AwayTeam.TLA,
        KickoffAt:       m.UtcDate,
        Status:          mapStatus(m.Status),
        Source:          "footballdata",
        FetchedAt:       time.Now().UTC(),
    }
    if m.Score.FullTime.Home != nil {
        r.HomeGoals = m.Score.FullTime.Home
        r.AwayGoals = m.Score.FullTime.Away
    }
    if m.Score.ExtraTime != nil && m.Score.ExtraTime.Home != nil {
        r.HomeGoalsAfterET = m.Score.ExtraTime.Home
        r.AwayGoalsAfterET = m.Score.ExtraTime.Away
    }
    if m.Score.Penalties != nil && m.Score.Penalties.Home != nil {
        r.HomeGoalsAfterPenalties = m.Score.Penalties.Home
        r.AwayGoalsAfterPenalties = m.Score.Penalties.Away
    }
    return r
}

func mapStatus(s string) string {
    switch s {
    case "FINISHED":
        return "finished"
    case "IN_PLAY", "PAUSED":
        return "in_progress"
    case "POSTPONED", "CANCELLED":
        return "cancelled"
    default:
        return "scheduled"
    }
}
```

## ChainedResultProvider (Composite + Chain of Responsibility)

```go
// internal/infrastructure/external/chained/result_provider.go
package chained

type ChainedResultProvider struct {
    primary  match.ResultProvider
    fallback match.ResultProvider
    logger   Logger
}

func NewChainedResultProvider(
    primary, fallback match.ResultProvider,
    logger Logger,
) *ChainedResultProvider {
    return &ChainedResultProvider{
        primary:  primary,
        fallback: fallback,
        logger:   logger,
    }
}

func (c *ChainedResultProvider) FetchResults(
    ctx context.Context,
    from, to time.Time,
) ([]match.ExternalResult, error) {
    results, err := c.primary.FetchResults(ctx, from, to)
    if err == nil {
        return results, nil
    }
    c.logger.Warnw("primary provider failed",
        "error", err,
        "fallback", "balldontlie")
    return c.fallback.FetchResults(ctx, from, to)
}
```

## Circuit Breaker

Usar `github.com/sony/gobreaker`:

```go
// internal/infrastructure/external/footballdata/breaker.go
package footballdata

import "github.com/sony/gobreaker"

func NewBreakerSettings() gobreaker.Settings {
    return gobreaker.Settings{
        Name:        "footballdata",
        MaxRequests: 1,
        Interval:    60 * time.Second,
        Timeout:     60 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 5
        },
    }
}

type BreakerAdapter struct {
    inner   match.ResultProvider
    breaker *gobreaker.CircuitBreaker
}

func WithBreaker(inner match.ResultProvider) *BreakerAdapter {
    return &BreakerAdapter{
        inner:   inner,
        breaker: gobreaker.NewCircuitBreaker(NewBreakerSettings()),
    }
}

func (b *BreakerAdapter) FetchResults(
    ctx context.Context,
    from, to time.Time,
) ([]match.ExternalResult, error) {
    v, err := b.breaker.Execute(func() (any, error) {
        return b.inner.FetchResults(ctx, from, to)
    })
    if err != nil {
        return nil, err
    }
    return v.([]match.ExternalResult), nil
}
```

## Mapeo de códigos de equipos

Cada fuente usa códigos distintos. La tabla `team_external_ids` resuelve esto:

```go
// internal/infrastructure/persistence/postgres/team_repo.go
func (r *TeamRepo) FindByExternalID(
    ctx context.Context,
    source, externalID string,
) (*team.Team, error) {
    row, err := r.q.GetTeamByExternalID(ctx, sqlc.GetTeamByExternalIDParams{
        Source:     source,
        ExternalID: externalID,
    })
    // ... mapping
}
```

```sql
-- queries/teams.sql
-- name: GetTeamByExternalID :one
SELECT t.*
FROM teams t
JOIN team_external_ids tei ON tei.team_id = t.id
WHERE tei.source = $1 AND tei.external_id = $2;
```

## Job de sincronización

```go
// internal/interfaces/jobs/result_sync_job.go
package jobs

type ResultSyncJob struct {
    provider     match.ResultProvider
    matches      match.Repository
    teams        team.Repository
    eventBus     EventBus
    logger       Logger
    clock        Clock
}

func (j *ResultSyncJob) Run(ctx context.Context) error {
    now := j.clock.Now()
    from := now.Add(-6 * time.Hour)
    to := now.Add(3 * time.Hour)

    results, err := j.provider.FetchResults(ctx, from, to)
    if err != nil {
        j.logger.Errorw("failed to fetch results", "error", err)
        return err
    }

    for _, ext := range results {
        if ext.Status != "finished" {
            continue
        }
        if err := j.processOne(ctx, ext); err != nil {
            j.logger.Errorw("failed to process result",
                "external_id", ext.ExternalMatchID,
                "error", err)
            // No abortar: seguir con los demás
        }
    }
    return nil
}

func (j *ResultSyncJob) processOne(ctx context.Context, ext match.ExternalResult) error {
    home, err := j.teams.FindByCode(ctx, ext.HomeTeamCode)
    if err != nil {
        return fmt.Errorf("home team %s not found: %w", ext.HomeTeamCode, err)
    }
    away, err := j.teams.FindByCode(ctx, ext.AwayTeamCode)
    if err != nil {
        return err
    }

    m, err := j.matches.FindByTeamsAndDate(ctx, home.ID(), away.ID(), ext.KickoffAt)
    if err != nil {
        return err
    }

    if m.Status() == match.StatusFinished {
        // Ya está finalizado. Solo actualizar si los goles difieren.
        if !m.ResultMatches(ext) {
            j.logger.Warnw("result differs from stored value, updating",
                "match_id", m.ID(), "stored", m.Result(), "external", ext)
            if err := m.OverrideResult(ext); err != nil {
                return err
            }
            if err := j.matches.Save(ctx, m); err != nil {
                return err
            }
        }
        return nil
    }

    if err := m.FinalizeWith(ext); err != nil {
        return err
    }
    if err := j.matches.Save(ctx, m); err != nil {
        return err
    }

    j.eventBus.Publish(ctx, events.MatchResultFinalized{
        MatchID: m.ID(),
        At:      j.clock.Now(),
    })
    return nil
}
```

## Tests con httptest

```go
// internal/infrastructure/external/footballdata/adapter_test.go
func TestAdapter_FetchResults_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "test-key", r.Header.Get("X-Auth-Token"))
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "matches": [{
                "id": 123,
                "utcDate": "2026-06-11T20:00:00Z",
                "status": "FINISHED",
                "homeTeam": {"id": 1, "tla": "MEX"},
                "awayTeam": {"id": 2, "tla": "FRA"},
                "score": {
                    "winner": "HOME_TEAM",
                    "duration": "REGULAR",
                    "fullTime": {"home": 2, "away": 1}
                }
            }]
        }`))
    }))
    defer server.Close()

    adapter := NewAdapter(NewClient(server.URL, "test-key"))
    results, err := adapter.FetchResults(context.Background(), time.Now(), time.Now())
    require.NoError(t, err)
    require.Len(t, results, 1)
    assert.Equal(t, "MEX", results[0].HomeTeamCode)
    assert.Equal(t, 2, *results[0].HomeGoals)
}

func TestAdapter_FetchResults_APIError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{"error": "invalid key"}`))
    }))
    defer server.Close()

    adapter := NewAdapter(NewClient(server.URL, "bad-key"))
    _, err := adapter.FetchResults(context.Background(), time.Now(), time.Now())
    assert.Error(t, err)
}
```

## Antipatrones

❌ **Hacer que el dominio dependa del DTO de la API.** El dominio define `ExternalResult`; el adapter mapea a él.

❌ **Exponer DTOs (`matchDTO`, etc) fuera del paquete del adapter.** Son detalle de implementación.

❌ **No timeout en el HTTP client.** Sin timeout puedes colgar el job indefinidamente.

❌ **Hacer retry en el adapter sin circuit breaker.** Reintentar contra una API caída te quema el rate limit.

❌ **Hardcodear endpoint base URLs.** Vienen de configuración (`.env`).
