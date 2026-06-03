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

type FormRepo struct {
	db *sql.DB
}

func NewFormRepo(db *sql.DB) *FormRepo {
	return &FormRepo{db: db}
}

// ── Public methods ────────────────────────────────────────────────────────────

func (r *FormRepo) ListByOwner(ctx context.Context, ownerID string, filter model.FormFilter) ([]model.Form, error) {
	where := []string{"f.owner_id = ?"}
	args := []any{ownerID}

	if filter.Query != "" {
		where = append(where, "LOWER(f.title) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
	}
	if filter.Status != "" {
		where = append(where, "f.status = ?")
		args = append(args, filter.Status)
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

	rows, err := r.db.QueryContext(ctx, q, args...)
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
	where := []string{
		"f.status = 'active'",
		"(f.expires_at IS NULL OR f.expires_at > NOW())",
	}
	args := []any{}

	if query != "" {
		where = append(where, "LOWER(f.title) LIKE ?")
		args = append(args, "%"+strings.ToLower(query)+"%")
	}

	q := fmt.Sprintf(`
		SELECT %s, COUNT(r.id) AS response_count
		FROM forms f
		LEFT JOIN form_responses r ON r.form_id = f.id
		WHERE %s
		GROUP BY f.id
		ORDER BY f.created_at DESC
	`, formCols("f"), strings.Join(where, " AND "))

	rows, err := r.db.QueryContext(ctx, q, args...)
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
	row := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT %s FROM forms f WHERE f.id = ?
	`, formCols("f")), id)

	f, err := scanForm(row)
	if err != nil {
		if err == sql.ErrNoRows {
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	themeJSON, _ := json.Marshal(req.Theme)
	formID := uuid.New().String()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO forms (id, owner_id, form_name, title, description, status,
		    starts_at, expires_at, max_responses,
		    limit_one_per_user, require_login, collect_email,
		    shuffle_questions, shuffle_options, show_individual_responses, quiz_enabled,
		    thank_you_message, theme)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, formID, ownerID, req.FormName, req.Title, req.Description, string(req.Status),
		req.StartsAt, req.ExpiresAt, req.MaxResponses,
		req.LimitOnePerUser, req.RequireLogin, req.CollectEmail,
		req.ShuffleQuestions, req.ShuffleOptions, req.ShowIndividualResponses, req.QuizEnabled,
		req.ThankYouMessage, themeJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("insert form: %w", err)
	}

	if err := insertSections(ctx, tx, formID, req.Sections); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, formID)
}

func (r *FormRepo) Update(ctx context.Context, id string, req model.FormRequest) (*model.Form, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	themeJSON, _ := json.Marshal(req.Theme)

	_, err = tx.ExecContext(ctx, `
		UPDATE forms SET
		    form_name=?, title=?, description=?, status=?,
		    starts_at=?, expires_at=?, max_responses=?,
		    limit_one_per_user=?, require_login=?, collect_email=?,
		    shuffle_questions=?, shuffle_options=?, show_individual_responses=?, quiz_enabled=?,
		    thank_you_message=?, theme=?, updated_at=NOW(6)
		WHERE id=?
	`, req.FormName, req.Title, req.Description, string(req.Status),
		req.StartsAt, req.ExpiresAt, req.MaxResponses,
		req.LimitOnePerUser, req.RequireLogin, req.CollectEmail,
		req.ShuffleQuestions, req.ShuffleOptions, req.ShowIndividualResponses, req.QuizEnabled,
		req.ThankYouMessage, themeJSON, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update form: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM form_sections WHERE form_id = ?`, id); err != nil {
		return nil, fmt.Errorf("delete sections: %w", err)
	}
	if err := insertSections(ctx, tx, id, req.Sections); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *FormRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM forms WHERE id = ?`, id)
	return err
}

func (r *FormRepo) Duplicate(ctx context.Context, id, ownerID string) (*model.Form, error) {
	src, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	req := model.FormRequest{
		FormName:                src.FormName,
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

// ── Section / question insertion ──────────────────────────────────────────────

func insertSections(ctx context.Context, tx *sql.Tx, formID string, sections []model.SectionInput) error {
	type entry struct {
		serverID string
		input    model.SectionInput
	}
	entries := make([]entry, len(sections))
	idMap := make(map[string]string)

	for i, sec := range sections {
		sID := uuid.New().String()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO form_sections (id, form_id, title, description, position, next_action)
			VALUES (?, ?, ?, ?, ?, '{"type":"next"}')
		`, sID, formID, sec.Title, sec.Description, i); err != nil {
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
			if _, err := tx.ExecContext(ctx, `
				UPDATE form_sections SET next_action = ? WHERE id = ?
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

func insertQuestion(ctx context.Context, tx *sql.Tx, sectionID string, pos int, q model.QuestionInput) (string, error) {
	lsJSON, _ := json.Marshal(q.LinearScale)
	valJSON, _ := json.Marshal(q.Validation)
	caJSON, _ := json.Marshal(q.CorrectAnswers)

	qID := uuid.New().String()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO questions
		    (id, section_id, type, title, description, required,
		     allow_file_upload, allow_other, branching_enabled, position,
		     linear_scale, validation, points, correct_answer, correct_answers, rating_max)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, qID, sectionID, string(q.Type), q.Title, q.Description, q.Required,
		q.AllowFileUpload, q.AllowOther, q.BranchingEnabled, pos,
		nullJSON(lsJSON), nullJSON(valJSON), q.Points, q.CorrectAnswer, nullJSON(caJSON), q.RatingMax,
	)
	if err != nil {
		return "", fmt.Errorf("insert question: %w", err)
	}
	return qID, nil
}

func insertOptions(ctx context.Context, tx *sql.Tx, questionID string, opts []model.OptionInput, idMap map[string]string) error {
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
		optID := uuid.New().String()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO question_options (id, question_id, text, position, go_to_type, go_to_section_id)
			VALUES (?, ?, ?, ?, ?, ?)
		`, optID, questionID, opt.Text, i, goToType, goToSectionID); err != nil {
			return fmt.Errorf("insert option: %w", err)
		}
	}
	return nil
}

// ── Section / question loading ────────────────────────────────────────────────

func (r *FormRepo) loadSections(ctx context.Context, f *model.Form) error {
	sRows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, position, next_action
		FROM form_sections WHERE form_id = ? ORDER BY position
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

	qRows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, section_id, type, title, description,
		       required, allow_file_upload, allow_other, branching_enabled, position,
		       linear_scale, validation, points, correct_answer, correct_answers, rating_max
		FROM questions WHERE section_id IN %s ORDER BY position
	`, inClause(len(sectionIDs))), stringsToAny(sectionIDs)...)
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

	oRows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, question_id, text, position, go_to_type, go_to_section_id
		FROM question_options WHERE question_id IN %s ORDER BY position
	`, inClause(len(questionIDs))), stringsToAny(questionIDs)...)
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

func formCols(alias string) string {
	a := alias + "."
	return fmt.Sprintf(`
		%sid, %sowner_id, %sform_name, %stitle, %sdescription, %sstatus,
		%sstarts_at, %sexpires_at, %smax_responses,
		%slimit_one_per_user, %srequire_login, %scollect_email,
		%sshuffle_questions, %sshuffle_options, %sshow_individual_responses, %squiz_enabled,
		%sthank_you_message, %stheme, %screated_at, %supdated_at`,
		a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a)
}

type scanner interface{ Scan(dest ...any) error }

func scanForm(row scanner) (*model.Form, error) {
	var f model.Form
	var themeJSON []byte
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.FormName, &f.Title, &f.Description, &f.Status,
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

func scanFormWithCount(row scanner) (*model.Form, error) {
	var f model.Form
	var themeJSON []byte
	if err := row.Scan(
		&f.ID, &f.OwnerID, &f.FormName, &f.Title, &f.Description, &f.Status,
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

func inClause(n int) string {
	if n == 0 {
		return "(NULL)"
	}
	return "(" + strings.Repeat("?,", n-1) + "?)"
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

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
