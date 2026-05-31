CREATE TABLE forms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',  -- draft | active | closed

    -- Response window
    starts_at   TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,

    -- Response limits
    max_responses       INT,
    limit_one_per_user  BOOLEAN NOT NULL DEFAULT FALSE,

    -- Access control
    require_login  BOOLEAN NOT NULL DEFAULT FALSE,
    collect_email  BOOLEAN NOT NULL DEFAULT FALSE,

    -- Behaviour
    shuffle_questions         BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_options           BOOLEAN NOT NULL DEFAULT FALSE,
    show_individual_responses BOOLEAN NOT NULL DEFAULT TRUE,
    quiz_enabled              BOOLEAN NOT NULL DEFAULT FALSE,

    -- Post-submit
    thank_you_message TEXT NOT NULL DEFAULT '',

    -- Theme (headerImageUrl, accentColor, fontFamily)
    theme JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_forms_owner_id ON forms(owner_id);
CREATE INDEX idx_forms_status    ON forms(status);
