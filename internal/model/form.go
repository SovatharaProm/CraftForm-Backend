package model

import "time"

type FormStatus string

const (
	FormStatusDraft  FormStatus = "draft"
	FormStatusActive FormStatus = "active"
	FormStatusClosed FormStatus = "closed"
)

type Theme struct {
	HeaderImageURL string `json:"headerImageUrl,omitempty"`
	AccentColor    string `json:"accentColor,omitempty"`
	FontFamily     string `json:"fontFamily,omitempty"`
}

type FormSection struct {
	ID          string        `json:"id"`
	FormID      string        `json:"formId"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Position    int           `json:"position"`
	NextAction  *BranchAction `json:"nextAction,omitempty"`
	Questions   []Question    `json:"questions"`
}

type Form struct {
	ID          string     `json:"id"`
	OwnerID     string     `json:"ownerId"`
	FormName    string     `json:"formName"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      FormStatus `json:"status"`

	StartsAt  *time.Time `json:"startsAt,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	MaxResponses    *int `json:"maxResponses,omitempty"`
	LimitOnePerUser bool `json:"limitOnePerUser"`
	RequireLogin    bool `json:"requireLogin"`
	CollectEmail    bool `json:"collectEmail"`

	ShuffleQuestions        bool `json:"shuffleQuestions"`
	ShuffleOptions          bool `json:"shuffleOptions"`
	ShowIndividualResponses bool `json:"showIndividualResponses"`
	QuizEnabled             bool `json:"quizEnabled"`

	ThankYouMessage string `json:"thankYouMessage,omitempty"`
	Theme           Theme  `json:"theme"`

	ResponseCount int           `json:"responseCount"`
	Sections      []FormSection `json:"sections"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type SectionInput struct {
	ClientID    string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	NextAction  *BranchAction   `json:"nextAction,omitempty"`
	Questions   []QuestionInput `json:"questions"`
}

type FormRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	FormName    string     `json:"formName"`
	Status      FormStatus `json:"status"`

	StartsAt  *time.Time `json:"startsAt,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	MaxResponses    *int `json:"maxResponses,omitempty"`
	LimitOnePerUser bool `json:"limitOnePerUser"`
	RequireLogin    bool `json:"requireLogin"`
	CollectEmail    bool `json:"collectEmail"`

	ShuffleQuestions        bool `json:"shuffleQuestions"`
	ShuffleOptions          bool `json:"shuffleOptions"`
	ShowIndividualResponses bool `json:"showIndividualResponses"`
	QuizEnabled             bool `json:"quizEnabled"`

	ThankYouMessage string         `json:"thankYouMessage"`
	Theme           Theme          `json:"theme"`
	Sections        []SectionInput `json:"sections"`
}

type FormFilter struct {
	Query  string // search by title
	Status string // draft | active | closed | "" (all)
	Sort   string // newest | oldest | most_responses | title
}
