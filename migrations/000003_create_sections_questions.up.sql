CREATE TABLE IF NOT EXISTS form_sections (
    id          CHAR(36)     NOT NULL PRIMARY KEY,
    form_id     CHAR(36)     NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    position    INT          NOT NULL DEFAULT 0,
    next_action JSON         NOT NULL,
    created_at  DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT fk_sections_form FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE
);

CREATE INDEX idx_form_sections_form_id ON form_sections(form_id);

CREATE TABLE IF NOT EXISTS questions (
    id          CHAR(36)     NOT NULL PRIMARY KEY,
    section_id  CHAR(36)     NOT NULL,

    type        VARCHAR(50)  NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    required            BOOLEAN NOT NULL DEFAULT FALSE,
    allow_file_upload   BOOLEAN NOT NULL DEFAULT FALSE,
    allow_other         BOOLEAN NOT NULL DEFAULT FALSE,
    branching_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    position    INT     NOT NULL DEFAULT 0,

    linear_scale JSON NULL,
    rating_max   INT  NULL,
    validation   JSON NULL,

    points          INT          NULL,
    correct_answer  VARCHAR(255) NOT NULL DEFAULT '',
    correct_answers JSON         NULL,

    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT fk_questions_section FOREIGN KEY (section_id) REFERENCES form_sections(id) ON DELETE CASCADE
);

CREATE INDEX idx_questions_section_id ON questions(section_id);

CREATE TABLE IF NOT EXISTS question_options (
    id          CHAR(36)     NOT NULL PRIMARY KEY,
    question_id CHAR(36)     NOT NULL,
    text        VARCHAR(255) NOT NULL DEFAULT '',
    position    INT          NOT NULL DEFAULT 0,

    go_to_type        VARCHAR(20) NULL,
    go_to_section_id  CHAR(36)    NULL,

    CONSTRAINT fk_options_question FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
    CONSTRAINT fk_options_section  FOREIGN KEY (go_to_section_id) REFERENCES form_sections(id) ON DELETE SET NULL
);

CREATE INDEX idx_question_options_question_id ON question_options(question_id);
