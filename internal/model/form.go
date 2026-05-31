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

	Sections  []FormSection `json:"sections"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// SectionInput holds a section payload from the frontend.
// ClientID is the frontend-generated ID used to resolve cross-section branching.
type SectionInput struct {
	ClientID    string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	NextAction  *BranchAction   `json:"nextAction,omitempty"`
	Questions   []QuestionInput `json:"questions"`
}

// FormRequest is shared for both create and update.
type FormRequest struct {
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

	ThankYouMessage string         `json:"thankYouMessage"`
	Theme           Theme          `json:"theme"`
	Sections        []SectionInput `json:"sections"`
}
