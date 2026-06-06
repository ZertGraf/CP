-- chapter 2: training analytics (gamification) + weather generator profiles + CWSI.
-- idempotent: also applied automatically at startup by storage.EnsureSchema.

ALTER TABLE sectors ADD COLUMN IF NOT EXISTS cwsi                   FLOAT NOT NULL DEFAULT 0;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS healthy_streak         INT   NOT NULL DEFAULT 0;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS safe_streak            INT   NOT NULL DEFAULT 0;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS crisis_streak          INT   NOT NULL DEFAULT 0;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS equipment_locked_ticks INT   NOT NULL DEFAULT 0;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS pest_active            BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE sectors ADD COLUMN IF NOT EXISTS last_alert_kind        VARCHAR(50) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS training_scores (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    session_id       UUID,
    total_score      FLOAT NOT NULL DEFAULT 0,
    badges           JSONB NOT NULL DEFAULT '[]',
    avg_health       FLOAT NOT NULL DEFAULT 0,
    water_efficiency FLOAT NOT NULL DEFAULT 0,
    sum_health       FLOAT NOT NULL DEFAULT 0,
    sum_efficiency   FLOAT NOT NULL DEFAULT 0,
    tick_count       INT NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS weather_configs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL DEFAULT 'default',
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    p_dry_to_wet FLOAT NOT NULL DEFAULT 0.20,
    p_wet_to_wet FLOAT NOT NULL DEFAULT 0.55,
    gamma_shape  FLOAT NOT NULL DEFAULT 1.5,
    gamma_scale  FLOAT NOT NULL DEFAULT 6.0,
    p_heat       FLOAT NOT NULL DEFAULT 0.05,
    p_pest_base  FLOAT NOT NULL DEFAULT 0.02,
    p_equipment  FLOAT NOT NULL DEFAULT 0.02,
    latitude     FLOAT NOT NULL DEFAULT 15.47,
    et_method    VARCHAR(20) NOT NULL DEFAULT 'hargreaves',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO weather_configs (name, is_active)
SELECT 'default', TRUE
WHERE NOT EXISTS (SELECT 1 FROM weather_configs);
