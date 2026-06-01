package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

func (r *FormRepo) ListByOwner(ctx context.Context, ownerID string, filter model.FormFilter) ([]model.Form, error) {
	args := []any{ownerID}
	where := []string{"f.owner_id = $1::uuid"}

	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
		where = append(where, fmt.Sprintf("LOWER(f.title) LIKE $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("f.status = $%d", len(args)))
	}

	orderBy := "f.created_at DESC"
	switch filter.Sort {
	case "oldest":
		orderBy = "f.created_at ASC"
	case "most_responses":
		orderBy = "response_count DESC"
	case "title":
		orderBy = "f.title ASC"
	}

	q := fmt.Sprintf(`
		SELECT %s, COUNT(r.id) AS response_count
		FROM forms f
		LEFT JOIN form_responses r ON r.form_id = f.id
		WHERE %s
		GROUP BY f.id
		ORDER BY %s
	`, formCols("f"), strings.Join(where, " AND "), orderBy)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []model.Form
	for rows.Next() {
		f, err := scanFormWithCount(rows)
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

func (r *FormRepo) ListPublic(ctx context.Context, query string) ([]model.Form, error) {
	args := []any{}
	where := []string{
		"f.status = 'active'",
		"(f.expires_at IS NULL OR f.expires_at > NOW())",
	}

	if query != "" {
		args = append(args, "%"+strings.ToLower(query)+"%")
		where = append(where, fmt.Sprintf("LOWER(f.title) LIKE $%d", len(args)))
	}

	q := fmt.Sprintf(`
		SELECT %s, COUNT(r.id) AS response_count
		FROM forms f
		LEFT JOIN form_responses r ON r.form_id = f.id
		WHERE %s
		GROUP BY f.id
		ORDER BY f.created_at DESC
	`, formCols("f"), strings.Join(where, " AND "))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []model.Form
	for rows.Next() {
		f, err := scanFormWithCount(rows)
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
	row := r.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT %s FROM forms f WHERE f.id = $1::uuid
	`, formCols("f")), id)

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
		INSERT INTO forms (owner_id, owner_name, title, description, status,
		    starts_at, expires_at, max_responses,
		    limit_one_per_user, require_login, collect_email,
		    shuffle_questions, shuffle_options, show_individual_responses, quiz_enabled,
		    thank_you_message, theme)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id::text
	`, ownerID, req.OwnerName, req.Title, req.Description, string(req.Status),
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
		    owner_name=$1, title=$2, description=$3, status=$4,
		    starts_at=$5, expires_at=$6, max_responses=$7,
		    limit_one_per_user=$8, require_login=$9, collect_email=$10,
		    shuffle_questions=$11, shuffle_options=$12, show_individual_responses=$13, quiz_enabled=$14,
		    thank_you_message=$15, theme=$16, updated_at=NOW()
		WHERE id=$17::uuid
	`, req.OwnerName, req.Title, req.Description, string(req.Status),
		req.StartsAt, req.ExpiresAt, req.MaxResponses,
		req.LimitOnePerUser, req.RequireLogin, req.CollectEmail,
		req.ShuffleQuestions, req.ShuffleOptions, req.ShowIndividualResponses, req.QuizEnabled,
		req.ThankYouMessage, themeJSON, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update form: %w", err)
	}

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
		OwnerName:               src.OwnerName,
		Title:                   "Copy of " + src.Title,
		Description:             src.Description,
		Status:                  model.FormStatusDraft,
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

// ── Section / question insertion (shared by Create and Update) ────────────────

func insertSections(ctx context.Context, tx pgx.Tx, formID string, sections []model.SectionInput) error {
	type entry struct {
		serverID string
		input    model.SectionInput
	}
	entries := make([]entry, len(sections))
	idMap := make(map[string]string)

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
		     linear_scale, validation, points, correct_answer, correct_answers, rating_max)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id::text
	`, sectionID, string(q.Type), q.Title, q.Description, q.Required,
		q.AllowFileUpload, q.AllowOther, q.BranchingEnabled, pos,
		nullJSON(lsJSON), nullJSON(valJSON), q.Points, q.CorrectAnswer, nullJSON(caJSON), q.RatingMax,
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
		if _, err := tx.Exec(ctx, `
			INSERT INTO question_options (question_id, text, position, go_to_type, go_to_section_id)
			VALUES ($1::uuid, $2, $3, $4, $5::uuid)
		`, questionID, opt.Text, i, goToType, goToSectionID); err != nil {
			return fmt.Errorf("insert option: %w", err)
		}
	}
	return nil
}

// ── Section / question loading ────────────────────────────────────────────────

func (r *FormRepo) loadSections(ctx context.Context, f *model.Form) error {
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

	qRows, err := r.pool.Query(ctx, `
		SELECT id::text, section_id::text, type, title, description,
		       required, allow_file_upload, allow_other, branching_enabled, position,
		       linear_scale, validation, points, correct_answer, correct_answers, rating_max
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
			&lsJSON, &valJSON, &q.Points, &q.CorrectAnswer, &caJSON, &q.RatingMax,
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

// ── Scan helpers ──────────────────────────────────────────────────────────────

// formCols returns the SELECT column list for the forms table aliased as `alias`.
func formCols(alias string) string {
	a := alias + "."
	return fmt.Sprintf(`
		%sid::text, %sowner_id::text, %sowner_name, %stitle, %sdescription, %sstatus,
		%sstarts_at, %sexpires_at, %smax_responses,
		%slimit_one_per_user, %srequire_login, %scollect_email,
		%sshuffle_questions, %sshuffle_options, %sshow_individual_responses, %squiz_enabled,
		%sthank_you_message, %stheme, %screated_at, %supdated_at`,
		a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a)
}

type scanner interface{ Scan(dest ...any) error }

// scanForm scans a form row without response_count.
func scanForm(row scanner) (*model.Form, error) {
	var f model.Form
	var themeJSON []byte
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.OwnerName, &f.Title, &f.Description, &f.Status,
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

// scanFormWithCount scans a form row that includes response_count as the last column.
func scanFormWithCount(row scanner) (*model.Form, error) {
	var f model.Form
	var themeJSON []byte
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.OwnerName, &f.Title, &f.Description, &f.Status,
		&f.StartsAt, &f.ExpiresAt, &f.MaxResponses,
		&f.LimitOnePerUser, &f.RequireLogin, &f.CollectEmail,
		&f.ShuffleQuestions, &f.ShuffleOptions, &f.ShowIndividualResponses, &f.QuizEnabled,
		&f.ThankYouMessage, &themeJSON,
		&f.CreatedAt, &f.UpdatedAt,
		&f.ResponseCount,
	); err != nil {
		return nil, err
	}
	if len(themeJSON) > 0 {
		json.Unmarshal(themeJSON, &f.Theme)
	}
	return &f, nil
}

// ── Utilities ─────────────────────────────────────────────────────────────────

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

func nullJSON(b []byte) interface{} {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
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
				LinearScale: q.LinearScale, RatingMax: q.RatingMax, Validation: q.Validation,
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
