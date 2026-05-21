# Quiniela Mundial 2026

Sistema de quinielas privadas para el Mundial de Fútbol 2026 con múltiples grupos, pronósticos por partido y por bracket completo del torneo.

## Stack tecnológico

- **Backend:** Go 1.22+ con Gin + sqlc
- **Base de datos:** PostgreSQL 16+
- **Frontend:** Angular 17+ standalone components
- **Auth:** JWT + invitaciones por email
- **APIs externas:** openfootball/worldcup.json + football-data.org + fallback manual

## Características principales

- **Multi-grupo privado:** crea cuántos grupos quieras, cada uno con sus integrantes
- **Doble pronóstico:** marcador por partido + bracket completo del torneo
- **Sistema de puntaje según reglas del usuario** (ver [SCORING-RULES.md](docs/SCORING-RULES.md))
- **3 paletas de colores** intercambiables (Mundial Vibrante, Dark Pro, Latina Cálida)
- **Co-administradores por grupo** con invitaciones por email
- **Transparencia post-cierre:** ver los pronósticos de los demás después del cierre

## Estructura del repositorio

```
quiniela-mundial/
├── docs/                          # Documentación de arquitectura y dominio
│   ├── ROADMAP.md                 # Plan por fases con fechas
│   ├── DOMAIN-MODEL.md            # Modelo de dominio
│   ├── ARCHITECTURE.md            # Clean Architecture aplicada
│   ├── SCORING-RULES.md           # Especificación formal de reglas
│   └── API-INTEGRATION.md         # Estrategia híbrida APIs externas
├── .claude/
│   ├── skills/                    # Skills para guiar a Claude Code
│   └── agents/                    # Subagentes especializados
└── frontend/
    └── styles/                    # Sistema de temas (3 paletas)
```

## Cómo empezar

1. Lee [docs/ROADMAP.md](docs/ROADMAP.md) para entender el plan por fases.
2. Lee [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) para entender las capas.
3. Lee [docs/DOMAIN-MODEL.md](docs/DOMAIN-MODEL.md) para entender el modelo de dominio.
4. Cuando vayas a codificar, los skills en `.claude/skills/` le dirán a Claude cómo construir cada parte.

## Fechas críticas

- **11 de junio de 2026** — Inicio del Mundial. Pronóstico de bracket se cierra al kickoff del partido inaugural.
- **19 de julio de 2026** — Final del Mundial.

## Licencia

Uso privado.
