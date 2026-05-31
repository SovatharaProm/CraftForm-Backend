CREATE TABLE form_sections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id     UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    position    INT  NOT NULL DEFAULT 0,
    -- {type: "next"|"submit"|"section", sectionId?: uuid-string}
    next_action JSONB NOT NULL DEFAULT '{"type":"next"}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_sections_form_id ON form_sections(form_id);

CREATE TABLE questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id  UUID NOT NULL REFERENCES form_sections(id) ON DELETE CASCADE,

    type        TEXT NOT NULL,  -- single|multiple|dropdown|short-text|paragraph|linear-scale|rating|date|time|file-upload
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    required            BOOLEAN NOT NULL DEFAULT FALSE,
    allow_file_upload   BOOLEAN NOT NULL DEFAULT FALSE,
    allow_other         BOOLEAN NOT NULL DEFAULT FALSE,
    branching_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    position    INT NOT NULL DEFAULT 0,

    -- Only populated for linear-scale / rating
    -- {min, max, minLabel?, maxLabel?}
    linear_scale JSONB,

    -- Validation rule: {type, min?, max?, pattern?, message?, start?, end?}
    validation JSONB,

    -- Quiz fields
    points          INT,
    correct_answer  TEXT NOT NULL DEFAULT '',
    correct_answers JSONB,   -- []string for multiple-answer quiz

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_section_id ON questions(section_id);

-- Only rows for question types: single, multiple, dropdown
CREATE TABLE question_options (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    text        TEXT NOT NULL DEFAULT '',
    position    INT  NOT NULL DEFAULT 0,

    -- Branching: which section to jump to when this option is selected
    go_to_type        TEXT,   -- "next" | "submit" | "section"
    go_to_section_id  UUID REFERENCES form_sections(id) ON DELETE SET NULL
);

CREATE INDEX idx_question_options_question_id ON question_options(question_id);
