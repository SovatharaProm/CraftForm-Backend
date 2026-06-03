CREATE TABLE IF NOT EXISTS form_responses (
    id               CHAR(36)     NOT NULL PRIMARY KEY,
    form_id          CHAR(36)     NOT NULL,
    user_id          CHAR(36)     NULL,

    respondent_name  VARCHAR(255) NOT NULL DEFAULT '',
    respondent_email VARCHAR(255) NOT NULL DEFAULT '',
    submitted_at     DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    score            INT          NULL,
    max_score        INT          NULL,

    CONSTRAINT fk_responses_form FOREIGN KEY (form_id) REFERENCES forms(id) ON DELETE CASCADE,
    CONSTRAINT fk_responses_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_form_responses_form_id ON form_responses(form_id);
CREATE INDEX idx_form_responses_user_id ON form_responses(user_id);

CREATE TABLE IF NOT EXISTS question_answers (
    id          CHAR(36) NOT NULL PRIMARY KEY,
    response_id CHAR(36) NOT NULL,
    question_id CHAR(36) NOT NULL,

    value       TEXT NULL,
    value_array JSON NULL,
    file_url    TEXT NULL,
    file_name   TEXT NULL,

    CONSTRAINT fk_answers_response FOREIGN KEY (response_id) REFERENCES form_responses(id) ON DELETE CASCADE,
    CONSTRAINT fk_answers_question FOREIGN KEY (question_id) REFERENCES questions(id)      ON DELETE CASCADE
);

CREATE INDEX idx_question_answers_response_id ON question_answers(response_id);
CREATE INDEX idx_question_answers_question_id ON question_answers(question_id);
