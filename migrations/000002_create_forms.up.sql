CREATE TABLE IF NOT EXISTS forms (
    id          CHAR(36)     NOT NULL PRIMARY KEY,
    owner_id    CHAR(36)     NOT NULL,

    form_name   VARCHAR(255) NOT NULL DEFAULT '',
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'draft',

    starts_at   DATETIME(6)  NULL,
    expires_at  DATETIME(6)  NULL,

    max_responses       INT     NULL,
    limit_one_per_user  BOOLEAN NOT NULL DEFAULT FALSE,

    require_login  BOOLEAN NOT NULL DEFAULT FALSE,
    collect_email  BOOLEAN NOT NULL DEFAULT FALSE,

    shuffle_questions         BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_options           BOOLEAN NOT NULL DEFAULT FALSE,
    show_individual_responses BOOLEAN NOT NULL DEFAULT TRUE,
    quiz_enabled              BOOLEAN NOT NULL DEFAULT FALSE,

    thank_you_message TEXT NOT NULL,

    theme JSON NOT NULL,

    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

    CONSTRAINT fk_forms_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_forms_owner_id ON forms(owner_id);
CREATE INDEX idx_forms_status   ON forms(status);
