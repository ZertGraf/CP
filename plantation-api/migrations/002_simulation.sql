CREATE TABLE telemetry (
                           id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                           sector_id   UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
                           soil_moisture FLOAT NOT NULL,
                           temperature   FLOAT NOT NULL,
                           health_index  FLOAT NOT NULL,
                           recorded_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_telemetry_sector_time ON telemetry (sector_id, recorded_at DESC);

CREATE TABLE notifications (
                               id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               sector_id  UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
                               user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
                               kind       VARCHAR(50) NOT NULL,
                               message    TEXT NOT NULL,
                               is_read    BOOLEAN NOT NULL DEFAULT FALSE,
                               created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE sectors ADD COLUMN IF NOT EXISTS last_watered_at TIMESTAMPTZ;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS daily_water_limit FLOAT NOT NULL DEFAULT 500;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS water_consumed FLOAT NOT NULL DEFAULT 0;