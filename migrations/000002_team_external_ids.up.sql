-- Add team_external_ids table for cross-API team code mapping

CREATE TABLE team_external_ids (
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    source      TEXT NOT NULL CHECK (source IN ('openfootball','footballdata','balldontlie')),
    external_id TEXT NOT NULL,
    PRIMARY KEY (team_id, source)
);

CREATE INDEX idx_team_external_ids_lookup ON team_external_ids(source, external_id);
