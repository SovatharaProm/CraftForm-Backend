CREATE TABLE form_responses (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id          UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id) ON DELETE SET NULL,  -- NULL for anonymous

    respondent_name  TEXT NOT NULL DEFAULT '',
    respondent_email TEXT NOT NULL DEFAULT '',
    submitted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    score            INT,
    max_score        INT
);

CREATE INDEX idx_form_responses_form_id ON form_responses(form_id);
CREATE INDEX idx_form_responses_user_id ON form_responses(user_id);

CREATE TABLE question_answers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    response_id UUID NOT NULL REFERENCES form_responses(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id)      ON DELETE CASCADE,

    value       TEXT,      -- single string: short-text, paragraph, date, time, single, linear-scale, rating
    value_array TEXT[],    -- multiple selection option IDs
    file_url    TEXT,      -- path returned from /api/upload
    file_name   TEXT       -- original filename for display
);

CREATE INDEX idx_question_answers_response_id ON question_answers(response_id);
CREATE INDEX idx_question_answers_question_id ON question_answers(question_id);
