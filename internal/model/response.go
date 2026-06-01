package model

import "time"

type FormResponse struct {
	ID              string           `json:"id"`
	FormID          string           `json:"formId"`
	UserID          *string          `json:"userId,omitempty"`
	RespondentName  string           `json:"respondentName"`
	RespondentEmail string           `json:"respondentEmail,omitempty"`
	SubmittedAt     time.Time        `json:"submittedAt"`
	Score           *int             `json:"score,omitempty"`
	MaxScore        *int             `json:"maxScore,omitempty"`
	Answers         []QuestionAnswer `json:"answers"`
}

type QuestionAnswer struct {
	ID         string   `json:"id"`
	ResponseID string   `json:"responseId"`
	QuestionID string   `json:"questionId"`
	Value      *string  `json:"value,omitempty"`
	ValueArray []string `json:"valueArray,omitempty"`
	FileURL    string   `json:"fileUrl,omitempty"`
	FileName   string   `json:"fileName,omitempty"`
}

type SubmitResponseRequest struct {
	RespondentName string        `json:"respondentName"`
	Answers        []AnswerInput `json:"answers"`
}

type AnswerInput struct {
	QuestionID string   `json:"questionId"`
	Value      *string  `json:"value,omitempty"`
	ValueArray []string `json:"valueArray,omitempty"`
	FileURL    string   `json:"fileUrl,omitempty"`
	FileName   string   `json:"fileName,omitempty"`
}

type ResponseFilter struct {
	Query    string
	DateFrom *time.Time
	DateTo   *time.Time
}

// ── Analytics ──────────────────────────────────────────────────────────────────

type FormAnalytics struct {
	TotalResponses int                  `json:"totalResponses"`
	AverageScore   *float64             `json:"averageScore,omitempty"`
	MaxScore       *int                 `json:"maxScore,omitempty"`
	Questions      []QuestionAnalytics  `json:"questions"`
}

type QuestionAnalytics struct {
	QuestionID    string             `json:"questionId"`
	Title         string             `json:"title"`
	Type          QuestionType       `json:"type"`
	TotalAnswered int                `json:"totalAnswered"`
	OptionCounts  []OptionCount      `json:"optionCounts,omitempty"`
	Average       *float64           `json:"average,omitempty"`
	Distribution  []DistributionItem `json:"distribution,omitempty"`
	TextResponses []string           `json:"textResponses,omitempty"`
}

type OptionCount struct {
	OptionID string  `json:"optionId"`
	Text     string  `json:"text"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
}

type DistributionItem struct {
	Value   string  `json:"value"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}
