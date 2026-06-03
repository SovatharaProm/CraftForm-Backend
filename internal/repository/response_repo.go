package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sovatharaprom/craftform-backend/internal/model"
)

type ResponseRepo struct {
	db *sql.DB
}

func NewResponseRepo(db *sql.DB) *ResponseRepo {
	return &ResponseRepo{db: db}
}

func (r *ResponseRepo) CountByFormID(ctx context.Context, formID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM form_responses WHERE form_id = ?`, formID,
	).Scan(&count)
	return count, err
}

func (r *ResponseRepo) UserAlreadySubmitted(ctx context.Context, formID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM form_responses
		WHERE form_id = ? AND user_id = ?
	`, formID, userID).Scan(&count)
	return count > 0, err
}

func (r *ResponseRepo) Create(
	ctx context.Context,
	formID string,
	userID *string,
	respondentEmail string,
	req model.SubmitResponseRequest,
	score, maxScore *int,
) (*model.FormResponse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	respID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO form_responses (id, form_id, user_id, respondent_name, respondent_email, score, max_score)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, respID, formID, userID, req.RespondentName, respondentEmail, score, maxScore)
	if err != nil {
		return nil, fmt.Errorf("insert response: %w", err)
	}

	for _, a := range req.Answers {
		var valueArrayJSON []byte
		if len(a.ValueArray) > 0 {
			valueArrayJSON, _ = json.Marshal(a.ValueArray)
		}
		ansID := uuid.New().String()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO question_answers (id, response_id, question_id, value, value_array, file_url, file_name)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, ansID, respID, a.QuestionID, a.Value, nullBytes(valueArrayJSON), nullStr(a.FileURL), nullStr(a.FileName)); err != nil {
			return nil, fmt.Errorf("insert answer: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, respID)
}

func (r *ResponseRepo) ListByFormID(ctx context.Context, formID string, filter model.ResponseFilter) ([]model.FormResponse, error) {
	where := []string{"form_id = ?"}
	args := []any{formID}

	if filter.Query != "" {
		where = append(where, "LOWER(respondent_name) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
	}
	if filter.Email != "" {
		where = append(where, "LOWER(respondent_email) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.Email)+"%")
	}
	if filter.DateFrom != nil {
		where = append(where, "submitted_at >= ?")
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != nil {
		where = append(where, "submitted_at <= ?")
		args = append(args, filter.DateTo)
	}

	q := fmt.Sprintf(`
		SELECT id, form_id, user_id, respondent_name, respondent_email,
		       submitted_at, score, max_score
		FROM form_responses
		WHERE %s
		ORDER BY submitted_at DESC
	`, strings.Join(where, " AND "))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []model.FormResponse
	var ids []string
	for rows.Next() {
		resp, err := scanResponse(rows)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
		ids = append(ids, resp.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []model.FormResponse{}, nil
	}

	if err := r.loadAnswers(ctx, responses, ids); err != nil {
		return nil, err
	}
	return responses, nil
}

func (r *ResponseRepo) GetByID(ctx context.Context, id string) (*model.FormResponse, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, form_id, user_id, respondent_name, respondent_email,
		       submitted_at, score, max_score
		FROM form_responses WHERE id = ?
	`, id)

	resp, err := scanResponse(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	aRows, err := r.db.QueryContext(ctx, `
		SELECT id, response_id, question_id,
		       value, value_array, COALESCE(file_url,''), COALESCE(file_name,'')
		FROM question_answers WHERE response_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()

	for aRows.Next() {
		a, err := scanAnswer(aRows)
		if err != nil {
			return nil, err
		}
		resp.Answers = append(resp.Answers, *a)
	}
	if resp.Answers == nil {
		resp.Answers = []model.QuestionAnswer{}
	}
	return resp, aRows.Err()
}

func (r *ResponseRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM form_responses WHERE id = ?`, id)
	return err
}

func (r *ResponseRepo) loadAnswers(ctx context.Context, responses []model.FormResponse, ids []string) error {
	idxByID := map[string]int{}
	for i, resp := range responses {
		idxByID[resp.ID] = i
		responses[i].Answers = []model.QuestionAnswer{}
	}

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, response_id, question_id,
		       value, value_array, COALESCE(file_url,''), COALESCE(file_name,'')
		FROM question_answers WHERE response_id IN %s
	`, inClause(len(ids))), stringsToAny(ids)...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		a, err := scanAnswer(rows)
		if err != nil {
			return err
		}
		if idx, ok := idxByID[a.ResponseID]; ok {
			responses[idx].Answers = append(responses[idx].Answers, *a)
		}
	}
	return rows.Err()
}

// ── Scan helpers ──────────────────────────────────────────────────────────────

func scanResponse(row scanner) (*model.FormResponse, error) {
	var r model.FormResponse
	var userID *string
	if err := row.Scan(
		&r.ID, &r.FormID, &userID, &r.RespondentName, &r.RespondentEmail,
		&r.SubmittedAt, &r.Score, &r.MaxScore,
	); err != nil {
		return nil, err
	}
	r.UserID = userID
	return &r, nil
}

func scanAnswer(row scanner) (*model.QuestionAnswer, error) {
	var a model.QuestionAnswer
	var valueArrayRaw []byte
	if err := row.Scan(
		&a.ID, &a.ResponseID, &a.QuestionID,
		&a.Value, &valueArrayRaw, &a.FileURL, &a.FileName,
	); err != nil {
		return nil, err
	}
	if len(valueArrayRaw) > 0 {
		json.Unmarshal(valueArrayRaw, &a.ValueArray)
	}
	return &a, nil
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
