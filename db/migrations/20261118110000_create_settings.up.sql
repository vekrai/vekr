CREATE TABLE settings (
    id BIGSERIAL PRIMARY KEY,
    locale TEXT NOT NULL DEFAULT 'en' CHECK (locale IN ('en', 'zh')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (locale) VALUES ('en');
