package model

import "time"

type FormResponse struct {
	ID              string           `json:"id"`
	FormID          string           `json:"formId"`
	UserID          *string          `json:"userId,omitempty"`
	RespondentName  string           `json:"respondentName"`
	RespondentEmail string           `json:"respondentEmail,omitempty"`
	SubmittedAt     time.Time        `json:"submittedAt"`
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
	RespondentName string                  `json:"respondentName"`
	Answers        []AnswerInput           `json:"answers"`
}

type AnswerInput struct {
	QuestionID string   `json:"questionId"`
	Value      *string  `json:"value,omitempty"`
	ValueArray []string `json:"valueArray,omitempty"`
	FileURL    string   `json:"fileUrl,omitempty"`
	FileName   string   `json:"fileName,omitempty"`
}
