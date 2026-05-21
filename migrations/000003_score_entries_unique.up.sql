ALTER TABLE score_entries ADD CONSTRAINT uq_score_entries UNIQUE (user_id, pool_id, source_type, source_ref);
