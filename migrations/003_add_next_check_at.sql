ALTER TABLE targets
    ADD COLUMN next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX idx_targets_next_check_at
    ON targets (next_check_at)
    WHERE active = TRUE;