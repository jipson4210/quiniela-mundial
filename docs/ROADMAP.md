# Roadmap del proyecto

> Hoy: **21 de mayo de 2026**
> Mundial arranca: **11 de junio de 2026**
> **Tiempo disponible: ~3 semanas**

El plan está apretado, así que las fases están priorizadas: lo crítico para que esté operativo al pitazo inicial está en las primeras fases. Lo opcional (notificaciones, pulido) puede quedar para después del Mundial empezado.

## Filosofía de priorización

**Lo que NO puede faltar al 11 de junio:**
1. Login + creación de grupo + invitación por email
2. Pronóstico de bracket (debe estar **cerrado** al kickoff inaugural)
3. Pronóstico de partido funcionando para fase de grupos
4. Cálculo de puntos al cerrar partido

**Lo que puede ir entrando en vivo:**
- Recálculo de bracket points (solo se necesita al cerrar grupos, ~26 de junio)
- UI bonita y temas (basta una vista funcional)
- Notificaciones
- Vista de pronósticos de los demás

---

## Fase 0 — Setup (3 días: 22-24 mayo)

**Objetivo:** repositorio funcionando, CI básico, Docker compose levantando todo.

- [ ] Init repo Git con esta estructura
- [ ] `go mod init` con módulos base (Gin, sqlc, pgx, viper, zerolog)
- [ ] PostgreSQL 16 en Docker Compose
- [ ] Migraciones con `golang-migrate`
- [ ] Skeleton de Clean Architecture (carpetas vacías con un placeholder)
- [ ] `make` con targets: `run`, `test`, `migrate`, `sqlc`
- [ ] Angular 17 init con standalone components
- [ ] Variables de entorno con `.env` + ejemplo `.env.example`

**Skill que usar:** `go-clean-architecture`

**Criterio de aceptación:** `docker compose up` levanta DB, API responde a `/health`, Angular sirve en `localhost:4200`.

---

## Fase 1 — Bootstrap de datos del Mundial (2 días: 25-26 mayo)

**Objetivo:** tener los 104 partidos y 48 equipos cargados en la base.

- [ ] Tablas `tournaments`, `teams`, `groups`, `matches`, `stages`
- [ ] Migración seed que descargue `worldcup.json` de openfootball
- [ ] Comando CLI `quiniela seed` que parsea y carga
- [ ] Endpoint `GET /api/v1/matches` para verificar

**Skill que usar:** `external-api-adapter`, `sqlc-repository`

**Criterio de aceptación:** la tabla `matches` tiene 104 filas, la tabla `teams` tiene 48 filas, agrupadas en 12 grupos.

---

## Fase 2 — Autenticación y grupos (3 días: 27-29 mayo)

**Objetivo:** usuarios pueden registrarse, crear grupos privados e invitar por email.

- [ ] Tabla `users` con email + password hash (bcrypt)
- [ ] Endpoint registro/login con JWT
- [ ] Tabla `pools` (grupos privados) con `creator_id`
- [ ] Tabla `pool_members` con roles (`creator`, `admin`, `member`)
- [ ] Endpoint crear grupo
- [ ] Tabla `invitations` con token único + expiración
- [ ] Endpoint invitar por email
- [ ] Servicio de email (SMTP o Resend API)
- [ ] Endpoint aceptar invitación con token

**Skill que usar:** `go-clean-architecture`, `domain-modeling-quiniela`

**Criterio de aceptación:** dos usuarios distintos pueden estar en el mismo grupo, donde uno es creator y otro miembro.

---

## Fase 3 — Pronósticos por partido (2 días: 30-31 mayo)

**Objetivo:** los usuarios pueden pronosticar marcadores de los 104 partidos.

- [ ] Tabla `match_predictions` con `(user_id, pool_id, match_id, home_goals, away_goals)`
- [ ] Endpoint crear/actualizar pronóstico
- [ ] **Validación clave:** rechazar si `now() >= match.kickoff_at - margen_de_cierre`
- [ ] Endpoint listar pronósticos del usuario en un grupo
- [ ] Tests unitarios del invariante de cierre

**Skill que usar:** `domain-modeling-quiniela`

**Criterio de aceptación:** un usuario puede meter pronósticos para los 104 partidos antes del 11 de junio. Después del kickoff de un partido específico, no se puede modificar ese pronóstico.

---

## Fase 4 — Pronóstico de bracket (3 días: 1-3 junio)

**Objetivo:** los usuarios pronostican qué equipos avanzan a cada fase + campeón + tercer puesto.

- [ ] Tabla `bracket_predictions` con:
  - 32 equipos a octavos
  - 16 equipos a cuartos
  - 8 equipos a semifinal
  - 4 equipos a final
  - 1 campeón
  - 1 tercer puesto
- [ ] Validación de coherencia: campeón ∈ finalistas ∈ semifinalistas ∈ cuartos ∈ octavos
- [ ] Endpoint guardar bracket
- [ ] **Validación clave:** rechazar si `now() >= tournament.kickoff_at`
- [ ] UI Angular para llenar el bracket
- [ ] Tests unitarios de invariantes de coherencia

**Skill que usar:** `bracket-prediction`

**Criterio de aceptación:** un usuario puede guardar su bracket completo antes del 11 de junio. Después del kickoff inaugural, el bracket queda congelado.

---

## Fase 5 — Motor de scoring (3 días: 4-6 junio)

**Objetivo:** calcular puntos automáticamente conforme se cargan resultados.

- [ ] Tabla `match_results` con `(match_id, home_goals, away_goals, finalized_at)`
- [ ] Tabla `score_entries` con `(user_id, pool_id, source_type, source_id, points)` para auditoría
- [ ] Implementar `MatchScoringStrategy` (3 + 1 + 1)
- [ ] Implementar `BracketScoringStrategy` (3/4/5/10/15/20 por fase)
- [ ] Implementar `ScoringEngine` con idempotencia (no duplicar puntos si se recalcula)
- [ ] Endpoint admin para cargar resultado manual
- [ ] Trigger automático al cargar resultado → recálculo
- [ ] Detector de avance de fase (cuando se completa grupos → calcula puntos de bracket de octavos, etc.)

**Skill que usar:** `scoring-strategy`
**Subagente:** `scoring-engine-agent`

**Criterio de aceptación:** dado un set de resultados de prueba, los puntos calculados coinciden con cálculo manual hecho en planilla.

---

## Fase 6 — Sync con API externa (2 días: 7-8 junio)

**Objetivo:** los resultados se actualizan automáticamente desde football-data.org.

- [ ] Adapter para `football-data.org` con interfaz `MatchResultProvider`
- [ ] Adapter para fallback a `balldontlie.io`
- [ ] Cron job cada 5 min durante días de partido
- [ ] Endpoint admin para sobrescribir resultado manualmente (fallback humano)
- [ ] Logs estructurados de cada sync

**Skill que usar:** `external-api-adapter`
**Subagente:** `fixture-sync-agent`

**Criterio de aceptación:** en una prueba con un partido jugado en la fecha, el resultado aparece en menos de 10 min sin intervención.

---

## Fase 7 — Ranking y vistas (3 días: 9-10 junio + buffer)

**Objetivo:** ver el ranking del grupo y los pronósticos de los demás.

- [ ] Endpoint ranking por grupo (suma de `score_entries` por usuario)
- [ ] Endpoint pronósticos de un partido finalizado (solo después de kickoff)
- [ ] Endpoint pronóstico de bracket de los demás (solo después del kickoff inaugural)
- [ ] UI Angular: tabla de ranking
- [ ] UI Angular: vista de pronósticos del grupo por partido
- [ ] UI Angular: comparativo de brackets

**Skill que usar:** `angular-feature`
**Subagente:** `ranking-builder-agent`

**Criterio de aceptación:** después de la primera jornada, el ranking refleja los puntos de todos los miembros.

---

## Fase 8 — Polish y temas (paralelo a Fase 7)

**Objetivo:** aplicación bonita y usable.

- [ ] Implementar 3 paletas con CSS custom properties
- [ ] Theme switcher en header
- [ ] Persistir tema en localStorage
- [ ] Responsive para móvil (la mayoría va a usar el celular)
- [ ] Loading states y manejo de errores

**Skill que usar:** `theme-system-3-palettes`, `angular-feature`

---

## Fase 9 — Post-Mundial empezado (en vivo)

**Objetivo:** mantener operación durante el torneo.

- [ ] Monitoreo de sync de resultados
- [ ] Soporte rápido a usuarios (canal Slack/WhatsApp del grupo)
- [ ] Hotfixes según aparezcan
- [ ] Notificaciones (recordatorio antes de kickoff) — si da tiempo

---

## Resumen de fechas

| Fase | Fechas | Días | Crítico |
|------|--------|------|---------|
| 0 — Setup | 22-24 mayo | 3 | Sí |
| 1 — Bootstrap datos | 25-26 mayo | 2 | Sí |
| 2 — Auth + grupos | 27-29 mayo | 3 | Sí |
| 3 — Pronóstico partido | 30-31 mayo | 2 | Sí |
| 4 — Pronóstico bracket | 1-3 junio | 3 | Sí |
| 5 — Motor de scoring | 4-6 junio | 3 | Sí |
| 6 — Sync API | 7-8 junio | 2 | Importante |
| 7 — Ranking + vistas | 9-10 junio | 2 | Sí |
| 8 — Polish + temas | paralelo | — | Opcional |
| **Cierre bracket** | **11 junio** | — | **Deadline** |

## Plan B si el tiempo aprieta

Si llegando al 5 de junio vas atrasado, **corta primero** estas piezas:
1. Co-administradores (deja solo creator) → simplifica permisos
2. Invitación por email (cambia a código de invitación copiable) → ahorra integración SMTP
3. Las 3 paletas (deja solo una) → ahorra theme system completo
4. Sync automático con API (carga manual los primeros días)

**Nunca cortes:** validaciones de cierre, motor de scoring, pronóstico de bracket.
