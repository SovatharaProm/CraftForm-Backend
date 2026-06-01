package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sovatharaprom/craftform-backend/internal/model"
)

type ResponseRepo struct {
	pool *pgxpool.Pool
}

func NewResponseRepo(pool *pgxpool.Pool) *ResponseRepo {
	return &ResponseRepo{pool: pool}
}

func (r *ResponseRepo) CountByFormID(ctx context.Context, formID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM form_responses WHERE form_id = $1::uuid`, formID,
	).Scan(&count)
	return count, err
}

func (r *ResponseRepo) UserAlreadySubmitted(ctx context.Context, formID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM form_responses
			WHERE form_id = $1::uuid AND user_id = $2::uuid
		)
	`, formID, userID).Scan(&exists)
	return exists, err
}

func (r *ResponseRepo) Create(
	ctx context.Context,
	formID string,
	userID *string,
	respondentEmail string,
	req model.SubmitResponseRequest,
	score, maxScore *int,
) (*model.FormResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var respID string
	err = tx.QueryRow(ctx, `
		INSERT INTO form_responses (form_id, user_id, respondent_name, respondent_email, score, max_score)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		RETURNING id::text
	`, formID, userID, req.RespondentName, respondentEmail, score, maxScore).Scan(&respID)
	if err != nil {
		return nil, fmt.Errorf("insert response: %w", err)
	}

	for _, a := range req.Answers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO question_answers (response_id, question_id, value, value_array, file_url, file_name)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		`, respID, a.QuestionID, a.Value, a.ValueArray, nullStr(a.FileURL), nullStr(a.FileName)); err != nil {
			return nil, fmt.Errorf("insert answer: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, respID)
}

func (r *ResponseRepo) ListByFormID(ctx context.Context, formID string, filter model.ResponseFilter) ([]model.FormResponse, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, form_id::text, user_id::text, respondent_name, respondent_email,
		       submitted_at, score, max_score
		FROM form_responses
		WHERE form_id = $1::uuid
		  AND ($2::text   IS NULL OR LOWER(respondent_name) LIKE $2)
		  AND ($3::timestamptz IS NULL OR submitted_at >= $3)
		  AND ($4::timestamptz IS NULL OR submitted_at <= $4)
		ORDER BY submitted_at DESC
	`, formID,
		likeOrNull(filter.Query),
		filter.DateFrom,
		filter.DateTo,
	)
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
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, form_id::text, user_id::text, respondent_name, respondent_email,
		       submitted_at, score, max_score
		FROM form_responses WHERE id = $1::uuid
	`, id)

	resp, err := scanResponse(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	aRows, err := r.pool.Query(ctx, `
		SELECT id::text, response_id::text, question_id::text,
		       value, value_array, COALESCE(file_url,''), COALESCE(file_name,'')
		FROM question_answers WHERE response_id = $1::uuid
	`, id)
	if err != nil {
		return nil, err
	}
	defer aRows.Close()

	for aRows.Next() {
		var a model.QuestionAnswer
		if err := aRows.Scan(&a.ID, &a.ResponseID, &a.QuestionID,
			&a.Value, &a.ValueArray, &a.FileURL, &a.FileName); err != nil {
			return nil, err
		}
		resp.Answers = append(resp.Answers, a)
	}
	if resp.Answers == nil {
		resp.Answers = []model.QuestionAnswer{}
	}
	return resp, aRows.Err()
}

func (r *ResponseRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM form_responses WHERE id = $1::uuid`, id)
	return err
}

// loadAnswers populates Answers on a slice of responses (batch fetch).
func (r *ResponseRepo) loadAnswers(ctx context.Context, responses []model.FormResponse, ids []string) error {
	idxByID := map[string]int{}
	for i, resp := range responses {
		idxByID[resp.ID] = i
		responses[i].Answers = []model.QuestionAnswer{}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id::text, response_id::text, question_id::text,
		       value, value_array, COALESCE(file_url,''), COALESCE(file_name,'')
		FROM question_answers WHERE response_id::text = ANY($1)
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var a model.QuestionAnswer
		if err := rows.Scan(&a.ID, &a.ResponseID, &a.QuestionID,
			&a.Value, &a.ValueArray, &a.FileURL, &a.FileName); err != nil {
			return err
		}
		if idx, ok := idxByID[a.ResponseID]; ok {
			responses[idx].Answers = append(responses[idx].Answers, a)
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

func likeOrNull(q string) *string {
	if q == "" {
		return nil
	}
	s := "%" + q + "%"
	return &s
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
