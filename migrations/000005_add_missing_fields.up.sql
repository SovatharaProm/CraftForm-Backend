-- owner_name: editable per-form display name (defaults to user's Google name)
ALTER TABLE forms ADD COLUMN owner_name TEXT NOT NULL DEFAULT '';

-- rating_max: max stars for rating questions (separate from linear_scale)
ALTER TABLE questions ADD COLUMN rating_max INT;

-- quiz score stored on submission so it doesn't need recalculation
ALTER TABLE form_responses ADD COLUMN score     INT;
ALTER TABLE form_responses ADD COLUMN max_score INT;
