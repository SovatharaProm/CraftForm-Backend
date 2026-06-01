ALTER TABLE form_responses DROP COLUMN IF EXISTS max_score;
ALTER TABLE form_responses DROP COLUMN IF EXISTS score;
ALTER TABLE questions      DROP COLUMN IF EXISTS rating_max;
ALTER TABLE forms          DROP COLUMN IF EXISTS owner_name;
