package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/sovatharaprom/craftform-backend/internal/model"
	"github.com/sovatharaprom/craftform-backend/internal/repository"
)

type ResponseService struct {
	responseRepo *repository.ResponseRepo
	formRepo     *repository.FormRepo
	userRepo     *repository.UserRepo
}

func NewResponseService(responseRepo *repository.ResponseRepo, formRepo *repository.FormRepo, userRepo *repository.UserRepo) *ResponseService {
	return &ResponseService{responseRepo: responseRepo, formRepo: formRepo, userRepo: userRepo}
}

func (s *ResponseService) Submit(ctx context.Context, formID string, userID *string, req model.SubmitResponseRequest) (*model.FormResponse, error) {
	form, err := s.formRepo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}

	// Gate checks
	if form.Status != model.FormStatusActive {
		return nil, model.ErrFormNotActive
	}
	if form.StartsAt != nil && time.Now().Before(*form.StartsAt) {
		return nil, model.ErrFormNotOpenYet
	}
	if form.ExpiresAt != nil && time.Now().After(*form.ExpiresAt) {
		return nil, model.ErrFormExpired
	}
	if form.RequireLogin && userID == nil {
		return nil, model.ErrUnauthorized
	}
	if form.MaxResponses != nil {
		count, err := s.responseRepo.CountByFormID(ctx, formID)
		if err != nil {
			return nil, err
		}
		if count >= *form.MaxResponses {
			return nil, model.ErrFormFull
		}
	}
	if form.LimitOnePerUser && userID != nil {
		already, err := s.responseRepo.UserAlreadySubmitted(ctx, formID, *userID)
		if err != nil {
			return nil, err
		}
		if already {
			return nil, model.ErrConflict
		}
	}

	// Collect respondent email when form requests it
	var respondentEmail string
	if form.CollectEmail {
		if userID != nil {
			if user, err := s.userRepo.GetByID(ctx, *userID); err == nil {
				respondentEmail = user.Email
			}
		} else if req.RespondentEmail != "" {
			respondentEmail = req.RespondentEmail
		}
	}

	// Quiz scoring
	var score, maxScore *int
	if form.QuizEnabled {
		s, m := calculateScore(form, req)
		score = &s
		maxScore = &m
	}

	return s.responseRepo.Create(ctx, formID, userID, respondentEmail, req, score, maxScore)
}

func (s *ResponseService) List(ctx context.Context, formID, ownerID string, filter model.ResponseFilter) ([]model.FormResponse, error) {
	if err := s.assertFormOwner(ctx, formID, ownerID); err != nil {
		return nil, err
	}
	return s.responseRepo.ListByFormID(ctx, formID, filter)
}

func (s *ResponseService) GetByID(ctx context.Context, responseID, formID, ownerID string) (*model.FormResponse, error) {
	if err := s.assertFormOwner(ctx, formID, ownerID); err != nil {
		return nil, err
	}
	return s.responseRepo.GetByID(ctx, responseID)
}

func (s *ResponseService) Delete(ctx context.Context, responseID, formID, ownerID string) error {
	if err := s.assertFormOwner(ctx, formID, ownerID); err != nil {
		return err
	}
	return s.responseRepo.Delete(ctx, responseID)
}

func (s *ResponseService) GetAnalytics(ctx context.Context, formID, ownerID string) (*model.FormAnalytics, error) {
	form, err := s.formRepo.GetByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form.OwnerID != ownerID {
		return nil, model.ErrForbidden
	}

	responses, err := s.responseRepo.ListByFormID(ctx, formID, model.ResponseFilter{})
	if err != nil {
		return nil, err
	}
	return buildAnalytics(form, responses), nil
}

func (s *ResponseService) assertFormOwner(ctx context.Context, formID, ownerID string) error {
	form, err := s.formRepo.GetByID(ctx, formID)
	if err != nil {
		return err
	}
	if form.OwnerID != ownerID {
		return model.ErrForbidden
	}
	return nil
}

// ── Quiz scoring ──────────────────────────────────────────────────────────────

func calculateScore(form *model.Form, req model.SubmitResponseRequest) (score, maxScore int) {
	answerMap := map[string]model.AnswerInput{}
	for _, a := range req.Answers {
		answerMap[a.QuestionID] = a
	}

	for _, sec := range form.Sections {
		for _, q := range sec.Questions {
			if q.Points == nil || *q.Points == 0 {
				continue
			}
			maxScore += *q.Points

			a, ok := answerMap[q.ID]
			if !ok {
				continue
			}
			switch q.Type {
			case model.QuestionTypeSingle, model.QuestionTypeDropdown:
				if a.Value != nil && *a.Value == q.CorrectAnswer {
					score += *q.Points
				}
			case model.QuestionTypeMultiple:
				if setsEqual(a.ValueArray, q.CorrectAnswers) {
					score += *q.Points
				}
			case model.QuestionTypeShortText:
				if a.Value != nil &&
					strings.EqualFold(strings.TrimSpace(*a.Value), strings.TrimSpace(q.CorrectAnswer)) {
					score += *q.Points
				}
			}
		}
	}
	return
}

func setsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

// ── Analytics aggregation ─────────────────────────────────────────────────────

func buildAnalytics(form *model.Form, responses []model.FormResponse) *model.FormAnalytics {
	analytics := &model.FormAnalytics{TotalResponses: len(responses)}

	// Build answer lookup: questionID → answers
	answersByQ := map[string][]model.QuestionAnswer{}
	var totalScore float64
	scoredResponses := 0
	for _, r := range responses {
		for _, a := range r.Answers {
			answersByQ[a.QuestionID] = append(answersByQ[a.QuestionID], a)
		}
		if r.Score != nil {
			totalScore += float64(*r.Score)
			scoredResponses++
		}
	}

	if form.QuizEnabled {
		maxScore := 0
		for _, sec := range form.Sections {
			for _, q := range sec.Questions {
				if q.Points != nil {
					maxScore += *q.Points
				}
			}
		}
		if maxScore > 0 {
			analytics.MaxScore = &maxScore
		}
		if scoredResponses > 0 {
			avg := totalScore / float64(scoredResponses)
			analytics.AverageScore = &avg
		}
	}

	// Build option text lookup: questionID → optionID → text
	optionText := map[string]map[string]string{}
	for _, sec := range form.Sections {
		for _, q := range sec.Questions {
			m := map[string]string{}
			for _, o := range q.Options {
				m[o.ID] = o.Text
			}
			optionText[q.ID] = m
		}
	}

	for _, sec := range form.Sections {
		for _, q := range sec.Questions {
			answers := answersByQ[q.ID]
			qa := model.QuestionAnalytics{
				QuestionID:    q.ID,
				Title:         q.Title,
				Type:          q.Type,
				TotalAnswered: len(answers),
			}
			total := len(responses)

			switch q.Type {
			case model.QuestionTypeSingle, model.QuestionTypeDropdown:
				counts := map[string]int{}
				for _, a := range answers {
					if a.Value != nil && *a.Value != "" {
						counts[*a.Value]++
					}
				}
				opts := optionText[q.ID]
				for optID, cnt := range counts {
					text := opts[optID]
					if text == "" {
						text = optID
					}
					pct := 0.0
					if total > 0 {
						pct = float64(cnt) / float64(total) * 100
					}
					qa.OptionCounts = append(qa.OptionCounts, model.OptionCount{
						OptionID: optID, Text: text, Count: cnt, Percent: pct,
					})
				}

			case model.QuestionTypeMultiple:
				counts := map[string]int{}
				for _, a := range answers {
					for _, v := range a.ValueArray {
						counts[v]++
					}
				}
				opts := optionText[q.ID]
				for optID, cnt := range counts {
					text := opts[optID]
					if text == "" {
						text = optID
					}
					pct := 0.0
					if total > 0 {
						pct = float64(cnt) / float64(total) * 100
					}
					qa.OptionCounts = append(qa.OptionCounts, model.OptionCount{
						OptionID: optID, Text: text, Count: cnt, Percent: pct,
					})
				}

			case model.QuestionTypeLinearScale, model.QuestionTypeRating:
				distCounts := map[string]int{}
				var sum float64
				validCount := 0
				for _, a := range answers {
					if a.Value != nil && *a.Value != "" {
						distCounts[*a.Value]++
						if v, err := strconv.ParseFloat(*a.Value, 64); err == nil {
							sum += v
							validCount++
						}
					}
				}
				for val, cnt := range distCounts {
					pct := 0.0
					if total > 0 {
						pct = float64(cnt) / float64(total) * 100
					}
					qa.Distribution = append(qa.Distribution, model.DistributionItem{
						Value: val, Count: cnt, Percent: pct,
					})
				}
				if validCount > 0 {
					avg := sum / float64(validCount)
					qa.Average = &avg
				}

			case model.QuestionTypeShortText, model.QuestionTypeParagraph,
				model.QuestionTypeDate, model.QuestionTypeTime:
				for _, a := range answers {
					if a.Value != nil && *a.Value != "" {
						qa.TextResponses = append(qa.TextResponses, *a.Value)
					}
				}
			}

			analytics.Questions = append(analytics.Questions, qa)
		}
	}
	return analytics
}
