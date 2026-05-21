-- Core tables for Quiniela Mundial 2026

-- uuid_generate_v7 via plpgsql (not in uuid-ossp on PG16)
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    unix_ts_ms bytea;
    uuid_bytes bytea;
    timestamp_bytes bytea;
BEGIN
    unix_ts_ms := int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint);
    uuid_bytes := gen_random_bytes(16);
    timestamp_bytes := overlay(uuid_bytes PLACING unix_ts_ms FROM 1 FOR 6);
    -- Set version 7 (0x70) and variant bits (0x80)
    timestamp_bytes := set_byte(timestamp_bytes, 6, (get_byte(timestamp_bytes, 6) & 15) | 112);
    timestamp_bytes := set_byte(timestamp_bytes, 8, (get_byte(timestamp_bytes, 8) & 63) | 128);
    RETURN encode(timestamp_bytes, 'hex')::uuid;
END;
$$;

-- Users
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL CHECK (char_length(display_name) BETWEEN 2 AND 50),
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tournaments
CREATE TABLE tournaments (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name       TEXT NOT NULL,
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_dates CHECK (starts_at < ends_at)
);

-- Teams
CREATE TABLE teams (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    code           TEXT NOT NULL UNIQUE CHECK (char_length(code) BETWEEN 3 AND 4),
    name           TEXT NOT NULL,
    flag_url       TEXT,
    confederation  TEXT,
    tournament_id  UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    UNIQUE (code, tournament_id)
);

-- Groups (World Cup groups: A through L)
CREATE TABLE stage_groups (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    name          TEXT NOT NULL CHECK (char_length(name) = 1),
    UNIQUE (tournament_id, name)
);

CREATE TABLE stage_group_teams (
    group_id UUID NOT NULL REFERENCES stage_groups(id) ON DELETE CASCADE,
    team_id  UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, team_id)
);

-- Stages
CREATE TABLE stages (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage         TEXT NOT NULL CHECK (stage IN ('group','round_of_32','round_of_16','quarter_final','semi_final','third_place','final')),
    points_per_correct_team INT NOT NULL DEFAULT 0,
    description   TEXT,
    UNIQUE (tournament_id, stage)
);

-- Matches
CREATE TABLE matches (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage         TEXT NOT NULL CHECK (stage IN ('group','round_of_32','round_of_16','quarter_final','semi_final','third_place','final')),
    group_id      UUID REFERENCES stage_groups(id),
    home_team_id  UUID NOT NULL REFERENCES teams(id),
    away_team_id  UUID NOT NULL REFERENCES teams(id),
    kickoff_at    TIMESTAMPTZ NOT NULL,
    venue         TEXT,
    status        TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled','in_progress','finished','cancelled')),
    home_goals        INT,
    away_goals        INT,
    home_goals_et     INT,
    away_goals_et     INT,
    home_goals_pen    INT,
    away_goals_pen    INT,
    result_source     TEXT CHECK (result_source IS NULL OR result_source IN ('api_footballdata','api_balldontlie','manual')),
    finalized_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_teams_different CHECK (home_team_id != away_team_id)
);

-- Pools (private groups)
CREATE TABLE pools (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name          TEXT NOT NULL CHECK (char_length(name) BETWEEN 3 AND 80),
    description   TEXT,
    creator_id    UUID NOT NULL REFERENCES users(id),
    tournament_id UUID NOT NULL REFERENCES tournaments(id),
    match_prediction_cutoff_minutes INT NOT NULL DEFAULT 0,
    extra_time_rule TEXT NOT NULL DEFAULT 'regular' CHECK (extra_time_rule IN ('regular','final_official')),
    show_other_predictions BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pool members
CREATE TABLE pool_members (
    pool_id   UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('creator','admin','member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    invited_by UUID REFERENCES users(id),
    PRIMARY KEY (pool_id, user_id)
);

-- Invitations
CREATE TABLE invitations (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pool_id    UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    invited_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Match predictions
CREATE TABLE match_predictions (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pool_id    UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    home_goals INT NOT NULL CHECK (home_goals >= 0 AND home_goals <= 30),
    away_goals INT NOT NULL CHECK (away_goals >= 0 AND away_goals <= 30),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, pool_id, match_id)
);

-- Bracket predictions
CREATE TABLE bracket_predictions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pool_id         UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    tournament_id   UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    teams_to_round_of_32 UUID[] NOT NULL,
    teams_to_round_of_16 UUID[] NOT NULL,
    teams_to_quarter_final UUID[] NOT NULL,
    teams_to_semi_final UUID[] NOT NULL,
    teams_to_final  UUID[] NOT NULL,
    third_place_winner UUID NOT NULL REFERENCES teams(id),
    champion        UUID NOT NULL REFERENCES teams(id),
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, pool_id)
);

-- Score entries (auditable points log)
CREATE TABLE score_entries (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pool_id     UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL CHECK (source_type IN ('match','bracket_stage','bracket_third','bracket_champion')),
    source_ref  TEXT NOT NULL,
    points      INT NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version     INT NOT NULL DEFAULT 1
);

CREATE INDEX idx_score_entries_pool_user ON score_entries(pool_id, user_id);
CREATE INDEX idx_match_predictions_match ON match_predictions(match_id);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_kickoff ON matches(kickoff_at);
