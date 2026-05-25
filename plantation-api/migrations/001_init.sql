CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('agronomist', 'operator');

CREATE TABLE users (
                       id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       email      VARCHAR(255) UNIQUE NOT NULL,
                       password   VARCHAR(255) NOT NULL,
                       name       VARCHAR(255) NOT NULL,
                       role       user_role NOT NULL DEFAULT 'operator',
                       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sectors (
                         id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                         name            VARCHAR(255) NOT NULL,
                         area_sqm        FLOAT NOT NULL DEFAULT 0,
                         soil_moisture   FLOAT NOT NULL DEFAULT 50,
                         temperature     FLOAT NOT NULL DEFAULT 25,
                         health_index    FLOAT NOT NULL DEFAULT 1.0,
                         gdd_cumulative  FLOAT NOT NULL DEFAULT 0,
                         phenophase      VARCHAR(10) NOT NULL DEFAULT '00',
                         ks_water        FLOAT NOT NULL DEFAULT 1.0,
                         ks_aeration     FLOAT NOT NULL DEFAULT 1.0,
                         deficit_dr      FLOAT NOT NULL DEFAULT 0,
                         status          VARCHAR(50) NOT NULL DEFAULT 'normal',
                         operator_id     UUID REFERENCES users(id) ON DELETE SET NULL,
                         created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                         updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE plants (
                        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        sector_id   UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
                        species     VARCHAR(255) NOT NULL DEFAULT 'litchi',
                        age_months  INT NOT NULL DEFAULT 0,
                        health      INT NOT NULL DEFAULT 100,
                        created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watering_logs (
                               id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               sector_id   UUID NOT NULL REFERENCES sectors(id) ON DELETE CASCADE,
                               user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               volume_liters FLOAT NOT NULL,
                               created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);