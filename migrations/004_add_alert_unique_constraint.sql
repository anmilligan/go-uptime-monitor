ALTER TABLE alerts
    ADD CONSTRAINT uq_alerts_incident_type
        UNIQUE (incident_id, type);