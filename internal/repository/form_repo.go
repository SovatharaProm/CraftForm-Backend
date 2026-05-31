package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sovatharaprom/craftform-backend/internal/model"
)

type FormRepo struct {
	pool *pgxpool.Pool
}

func NewFormRepo(pool *pgxpool.Pool) *FormRepo {
	return &FormRepo{pool: pool}
}

// ── Public methods ────────────────────────────────────────────────────────────

func (r *FormRepo) ListByOwner(ctx context.Context, ownerID string) ([]model.Form, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, owner_id::text, title, description, status,
		       starts_at, expires_at, max_responses,
		       limit_one_per_user, require_login, collect_email,
		       shuffle_questions, shuffle_options, show_individual_responses, quiz_enabled,
		       thank_you_message, theme, created_at, updated_at
		FROM forms
		WHERE owner_id = $1::uuid
		ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []model.Form
	for rows.Next() {
		f, err := scanForm(rows)
		if err != nil {
			return nil, err
		}
		f.Sections = []model.FormSection{}
		forms = append(forms, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if forms == nil {
		forms = []model.Form{}
	}
	return forms, nil
}

func (r *FormRepo) GetByID(ctx context.Context, id string) (*model.Form, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, owner_id::text, title, description, status,
		       starts_at, expires_at, max_responses,
		       limit_one_per_user, require_login, collect_email,
		       shuffle_questions, shuffle_options, show_individual_responses, quiz_enabled,
		       thank_you_message, theme, created_at, updated_at
		FROM forms WHERE id = $1::uuid
	`, id)

	f, err := scanForm(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	if err := r.loadSections(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (r *FormRepo) Create(ctx context.Context, ownerID string, req model.FormRequest) (*model.Form, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	themeJSON, _ := json.Marshal(req.Theme)

	var formID string
	err = tx.QueryRow(ctx, `
		INSERT INTO forms (owner_id, title, description, status,
		    starts_at, expires_at, max_responses,
		    limit_one_per_user, require_login, collect_email,
		    shuffle_questions, shuffle_options, show_individual_responses, quiz_enabled,
		    thank_you_message, theme)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id::text
	`, ownerID, req.Title, req.Description, string(req.Status),
		req.StartsAt, req.ExpiresAt, req.MaxResponses,
		req.LimitOnePerUser, req.RequireLogin, req.CollectEmail,
		req.ShuffleQuestions, req.ShuffleOptions, req.ShowIndividualResponses, req.QuizEnabled,
		req.ThankYouMessage, themeJSON,
	).Scan(&formID)
	if err != nil {
		return nil, fmt.Errorf("insert form: %w", err)
	}

	if err := insertSections(ctx, tx, formID, req.Sections); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, formID)
}

func (r *FormRepo) Update(ctx context.Context, id string, req model.FormRequest) (*model.Form, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	themeJSON, _ := json.Marshal(req.Theme)

	_, err = tx.Exec(ctx, `
		UPDATE forms SET
		    title=$1, description=$2, status=$3,
		    starts_at=$4, expires_at=$5, max_responses=$6,
		    limit_one_per_user=$7, require_login=$8, collect_email=$9,
		    shuffle_questions=$10, shuffle_options=$11, show_individual_responses=$12, quiz_enabled=$13,
		    thank_you_message=$14, theme=$15, updated_at=NOW()
		WHERE id=$16::uuid
	`, req.Title, req.Description, string(req.Status),
		req.StartsAt, req.ExpiresAt, req.MaxResponses,
		req.LimitOnePerUser, req.RequireLogin, req.CollectEmail,
		req.ShuffleQuestions, req.ShuffleOptions, req.ShowIndividualResponses, req.QuizEnabled,
		req.ThankYouMessage, themeJSON, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update form: %w", err)
	}

	// Delete existing sections — CASCADE removes questions and options
	if _, err = tx.Exec(ctx, `DELETE FROM form_sections WHERE form_id = $1::uuid`, id); err != nil {
		return nil, fmt.Errorf("delete sections: %w", err)
	}

	if err := insertSections(ctx, tx, id, req.Sections); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *FormRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM forms WHERE id = $1::uuid`, id)
	return err
}

func (r *FormRepo) Duplicate(ctx context.Context, id, ownerID string) (*model.Form, error) {
	src, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	req := model.FormRequest{
		Title:                   "Copy of " + src.Title,
		Description:             src.Description,
		Status:                  model.FormStatusDraft,
		StartsAt:                src.StartsAt,
		ExpiresAt:               src.ExpiresAt,
		MaxResponses:            src.MaxResponses,
		LimitOnePerUser:         src.LimitOnePerUser,
		RequireLogin:            src.RequireLogin,
		CollectEmail:            src.CollectEmail,
		ShuffleQuestions:        src.ShuffleQuestions,
		ShuffleOptions:          src.ShuffleOptions,
		ShowIndividualResponses: src.ShowIndividualResponses,
		QuizEnabled:             src.QuizEnabled,
		ThankYouMessage:         src.ThankYouMessage,
		Theme:                   src.Theme,
		Sections:                formSectionsToInput(src.Sections),
	}
	return r.Create(ctx, ownerID, req)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// insertSections handles the two-pass insert:
//  1. Insert all sections to obtain server UUIDs, building clientID→serverUUID map.
//  2. Update next_action / option go_to_section_id using the resolved map.
func insertSections(ctx context.Context, tx pgx.Tx, formID string, sections []model.SectionInput) error {
	type entry struct {
		serverID string
		input    model.SectionInput
	}
	entries := make([]entry, len(sections))
	idMap := make(map[string]string) // clientID → serverUUID

	// Pass 1: insert sections with placeholder next_action
	for i, sec := range sections {
		var sID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO form_sections (form_id, title, description, position, next_action)
			VALUES ($1::uuid, $2, $3, $4, '{"type":"next"}'::jsonb)
			RETURNING id::text
		`, formID, sec.Title, sec.Description, i).Scan(&sID); err != nil {
			return fmt.Errorf("insert section: %w", err)
		}
		entries[i] = entry{serverID: sID, input: sec}
		if sec.ClientID != "" {
			idMap[sec.ClientID] = sID
		}
	}

	// Pass 2: update next_action with resolved IDs, insert questions + options
	for _, e := range entries {
		if e.input.NextAction != nil {
			naJSON, _ := json.Marshal(resolveAction(e.input.NextAction, idMap))
			if _, err := tx.Exec(ctx, `
				UPDATE form_sections SET next_action = $1 WHERE id = $2::uuid
			`, naJSON, e.serverID); err != nil {
				return fmt.Errorf("update next_action: %w", err)
			}
		}

		for qi, q := range e.input.Questions {
			qID, err := insertQuestion(ctx, tx, e.serverID, qi, q)
			if err != nil {
				return err
			}
			if model.IsOptionQuestion(q.Type) {
				if err := insertOptions(ctx, tx, qID, q.Options, idMap); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func insertQuestion(ctx context.Context, tx pgx.Tx, sectionID string, pos int, q model.QuestionInput) (string, error) {
	lsJSON, _ := json.Marshal(q.LinearScale)
	valJSON, _ := json.Marshal(q.Validation)
	caJSON, _ := json.Marshal(q.CorrectAnswers)

	var qID string
	err := tx.QueryRow(ctx, `
		INSERT INTO questions
		    (section_id, type, title, description, required,
		     allow_file_upload, allow_other, branching_enabled, position,
		     linear_scale, validation, points, correct_answer, correct_answers)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id::text
	`, sectionID, string(q.Type), q.Title, q.Description, q.Required,
		q.AllowFileUpload, q.AllowOther, q.BranchingEnabled, pos,
		nullJSON(lsJSON), nullJSON(valJSON), q.Points, q.CorrectAnswer, nullJSON(caJSON),
	).Scan(&qID)
	if err != nil {
		return "", fmt.Errorf("insert question: %w", err)
	}
	return qID, nil
}

func insertOptions(ctx context.Context, tx pgx.Tx, questionID string, opts []model.OptionInput, idMap map[string]string) error {
	for i, opt := range opts {
		var goToType, goToSectionID *string
		if opt.GoTo != nil {
			goToType = &opt.GoTo.Type
			if opt.GoTo.SectionID != "" {
				if resolved, ok := idMap[opt.GoTo.SectionID]; ok {
					goToSectionID = &resolved
				} else {
					goToSectionID = &opt.GoTo.SectionID
				}
			}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO question_options (question_id, text, position, go_to_type, go_to_section_id)
			VALUES ($1::uuid, $2, $3, $4, $5::uuid)
		`, questionID, opt.Text, i, goToType, goToSectionID)
		if err != nil {
			return fmt.Errorf("insert option: %w", err)
		}
	}
	return nil
}

// loadSections fetches sections → questions → options in 3 batch queries and assembles the tree.
func (r *FormRepo) loadSections(ctx context.Context, f *model.Form) error {
	// 1. Sections
	sRows, err := r.pool.Query(ctx, `
		SELECT id::text, title, description, position, next_action
		FROM form_sections WHERE form_id = $1::uuid ORDER BY position
	`, f.ID)
	if err != nil {
		return err
	}
	defer sRows.Close()

	sectionIdx := map[string]int{}
	var sectionIDs []string

	for sRows.Next() {
		var s model.FormSection
		var naJSON []byte
		if err := sRows.Scan(&s.ID, &s.Title, &s.Description, &s.Position, &naJSON); err != nil {
			return err
		}
		s.FormID = f.ID
		s.Questions = []model.Question{}
		if len(naJSON) > 0 {
			json.Unmarshal(naJSON, &s.NextAction)
		}
		sectionIdx[s.ID] = len(f.Sections)
		sectionIDs = append(sectionIDs, s.ID)
		f.Sections = append(f.Sections, s)
	}
	if err := sRows.Err(); err != nil {
		return err
	}
	if len(sectionIDs) == 0 {
		f.Sections = []model.FormSection{}
		return nil
	}

	// 2. Questions (batch by section IDs)
	qRows, err := r.pool.Query(ctx, `
		SELECT id::text, section_id::text, type, title, description,
		       required, allow_file_upload, allow_other, branching_enabled, position,
		       linear_scale, validation, points, correct_answer, correct_answers
		FROM questions WHERE section_id::text = ANY($1) ORDER BY position
	`, sectionIDs)
	if err != nil {
		return err
	}
	defer qRows.Close()

	type qRef struct{ sIdx, qIdx int }
	qRefs := map[string]qRef{}
	var questionIDs []string

	for qRows.Next() {
		var q model.Question
		var lsJSON, valJSON, caJSON []byte
		if err := qRows.Scan(
			&q.ID, &q.SectionID, &q.Type, &q.Title, &q.Description,
			&q.Required, &q.AllowFileUpload, &q.AllowOther, &q.BranchingEnabled, &q.Position,
			&lsJSON, &valJSON, &q.Points, &q.CorrectAnswer, &caJSON,
		); err != nil {
			return err
		}
		q.Options = []model.AnswerOption{}
		if len(lsJSON) > 0 {
			json.Unmarshal(lsJSON, &q.LinearScale)
		}
		if len(valJSON) > 0 {
			json.Unmarshal(valJSON, &q.Validation)
		}
		if len(caJSON) > 0 {
			json.Unmarshal(caJSON, &q.CorrectAnswers)
		}

		sIdx := sectionIdx[q.SectionID]
		f.Sections[sIdx].Questions = append(f.Sections[sIdx].Questions, q)
		qIdx := len(f.Sections[sIdx].Questions) - 1
		qRefs[q.ID] = qRef{sIdx: sIdx, qIdx: qIdx}
		questionIDs = append(questionIDs, q.ID)
	}
	if err := qRows.Err(); err != nil {
		return err
	}
	if len(questionIDs) == 0 {
		return nil
	}

	// 3. Options (batch by question IDs)
	oRows, err := r.pool.Query(ctx, `
		SELECT id::text, question_id::text, text, position, go_to_type, go_to_section_id::text
		FROM question_options WHERE question_id::text = ANY($1) ORDER BY position
	`, questionIDs)
	if err != nil {
		return err
	}
	defer oRows.Close()

	for oRows.Next() {
		var opt model.AnswerOption
		var qID string
		var goToType, goToSectionID *string
		if err := oRows.Scan(&opt.ID, &qID, &opt.Text, &opt.Position, &goToType, &goToSectionID); err != nil {
			return err
		}
		if goToType != nil {
			opt.GoTo = &model.BranchAction{Type: *goToType}
			if goToSectionID != nil {
				opt.GoTo.SectionID = *goToSectionID
			}
		}
		if ref, ok := qRefs[qID]; ok {
			f.Sections[ref.sIdx].Questions[ref.qIdx].Options = append(
				f.Sections[ref.sIdx].Questions[ref.qIdx].Options, opt,
			)
		}
	}
	return oRows.Err()
}

// ── Utility ───────────────────────────────────────────────────────────────────

func resolveAction(a *model.BranchAction, idMap map[string]string) *model.BranchAction {
	if a == nil {
		return nil
	}
	out := *a
	if a.SectionID != "" {
		if resolved, ok := idMap[a.SectionID]; ok {
			out.SectionID = resolved
		}
	}
	return &out
}

// nullJSON returns nil if the marshalled value is "null", otherwise the bytes.
func nullJSON(b []byte) interface{} {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
}

// scanForm scans one form row (without sections).
type scanner interface {
	Scan(dest ...any) error
}

func scanForm(row scanner) (*model.Form, error) {
	var f model.Form
	var themeJSON []byte
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.Title, &f.Description, &f.Status,
		&f.StartsAt, &f.ExpiresAt, &f.MaxResponses,
		&f.LimitOnePerUser, &f.RequireLogin, &f.CollectEmail,
		&f.ShuffleQuestions, &f.ShuffleOptions, &f.ShowIndividualResponses, &f.QuizEnabled,
		&f.ThankYouMessage, &themeJSON,
		&f.CreatedAt, &f.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(themeJSON) > 0 {
		json.Unmarshal(themeJSON, &f.Theme)
	}
	return &f, nil
}

func formSectionsToInput(sections []model.FormSection) []model.SectionInput {
	out := make([]model.SectionInput, len(sections))
	for i, s := range sections {
		qs := make([]model.QuestionInput, len(s.Questions))
		for j, q := range s.Questions {
			opts := make([]model.OptionInput, len(q.Options))
			for k, o := range q.Options {
				opts[k] = model.OptionInput{Text: o.Text, GoTo: o.GoTo}
			}
			qs[j] = model.QuestionInput{
				ClientID: q.ID, Type: q.Type, Title: q.Title, Description: q.Description,
				Required: q.Required, AllowFileUpload: q.AllowFileUpload, AllowOther: q.AllowOther,
				BranchingEnabled: q.BranchingEnabled, Options: opts,
				LinearScale: q.LinearScale, Validation: q.Validation,
				Points: q.Points, CorrectAnswer: q.CorrectAnswer, CorrectAnswers: q.CorrectAnswers,
			}
		}
		out[i] = model.SectionInput{
			ClientID: s.ID, Title: s.Title, Description: s.Description,
			NextAction: s.NextAction, Questions: qs,
		}
	}
	return out
}
