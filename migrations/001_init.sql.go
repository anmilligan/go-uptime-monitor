package migrations


CREATE TABLE users (
	id BIGSERIAL PRIMARY KEY,
	telegram_id BIGINT NOT NULL UNIQUE,
	username TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

CREATE TABLE targets (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL
	REFERENCES users(id)
	ON DELETE CASCADE,

	kind TEXT NOT NULL,
	address TEXT NOT NULL,

	interval_seconds BIGINT NOT NULL,
	timeout_seconds BIGINT NOT NULL,

	expected_status INT,
	expected_body TEXT,

	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

	CHECK (kind IN ('http', 'tcp'))
)

CREATE TABLE checks (
	id BIGSERIAL PRIMARY KEY,

	target_id BIGINT NOT NULL
	REFERENCES targets(id)
	ON DELETE CASCADE,

	status INT NOT NULL,
	answer TEXT NOT NULL,

	response_time_ms BIGINT NOT NULL,
	is_success BOOLEAN NOT NULL,

	checked_at TIMESTAMPTZ NOT NULL
)

CREATE TABLE incidents (
	id BIGSERIAL PRIMARY KEY,

	target_id BIGINT NOT NULL
	REFERENCES targets(id)
	ON DELETE CASCADE,

	status TEXT NOT NULL,
	CHECK (status IN ('OPEN', 'CLOSED')),

	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	closed_at TIMESTAMPTZ
)

CREATE TABLE alerts (
	id BIGSERIAL PRIMARY KEY,

	incident_id BIGINT NOT NULL
	REFERENCES incidents(id)
	ON DELETE CASCADE,

	type TEXT NOT NULL,
	message TEXT NOT NULL,

	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

