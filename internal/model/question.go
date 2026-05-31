package model

type QuestionType string

const (
	QuestionTypeSingle      QuestionType = "single"
	QuestionTypeMultiple    QuestionType = "multiple"
	QuestionTypeDropdown    QuestionType = "dropdown"
	QuestionTypeLinearScale QuestionType = "linear-scale"
	QuestionTypeRating      QuestionType = "rating"
	QuestionTypeShortText   QuestionType = "short-text"
	QuestionTypeParagraph   QuestionType = "paragraph"
	QuestionTypeDate        QuestionType = "date"
	QuestionTypeTime        QuestionType = "time"
	QuestionTypeFileUpload  QuestionType = "file-upload"
)

func IsOptionQuestion(t QuestionType) bool {
	return t == QuestionTypeSingle || t == QuestionTypeMultiple || t == QuestionTypeDropdown
}

type BranchAction struct {
	Type      string `json:"type"`
	SectionID string `json:"sectionId,omitempty"`
}

type AnswerOption struct {
	ID       string        `json:"id"`
	Text     string        `json:"text"`
	Position int           `json:"position"`
	GoTo     *BranchAction `json:"goTo,omitempty"`
}

type LinearScale struct {
	Min      int    `json:"min"`
	Max      int    `json:"max"`
	MinLabel string `json:"minLabel,omitempty"`
	MaxLabel string `json:"maxLabel,omitempty"`
}

type ValidationRule struct {
	Type    string `json:"type"`
	Min     *int   `json:"min,omitempty"`
	Max     *int   `json:"max,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Message string `json:"message,omitempty"`
	Start   string `json:"start,omitempty"`
	End     string `json:"end,omitempty"`
}

type Question struct {
	ID               string          `json:"id"`
	SectionID        string          `json:"sectionId"`
	Type             QuestionType    `json:"type"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Required         bool            `json:"required"`
	AllowFileUpload  bool            `json:"allowFileUpload"`
	AllowOther       bool            `json:"allowOther"`
	BranchingEnabled bool            `json:"branchingEnabled"`
	Position         int             `json:"position"`
	Options          []AnswerOption  `json:"options"`
	LinearScale      *LinearScale    `json:"linearScale,omitempty"`
	Validation       *ValidationRule `json:"validation,omitempty"`
	Points           *int            `json:"points,omitempty"`
	CorrectAnswer    string          `json:"correctAnswer,omitempty"`
	CorrectAnswers   []string        `json:"correctAnswers,omitempty"`
}

// QuestionInput is used in create/update requests.
// ClientID holds the frontend-generated ID used to resolve branching references.
type QuestionInput struct {
	ClientID         string          `json:"id"`
	Type             QuestionType    `json:"type"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Required         bool            `json:"required"`
	AllowFileUpload  bool            `json:"allowFileUpload"`
	AllowOther       bool            `json:"allowOther"`
	BranchingEnabled bool            `json:"branchingEnabled"`
	Options          []OptionInput   `json:"options"`
	LinearScale      *LinearScale    `json:"linearScale,omitempty"`
	Validation       *ValidationRule `json:"validation,omitempty"`
	Points           *int            `json:"points,omitempty"`
	CorrectAnswer    string          `json:"correctAnswer,omitempty"`
	CorrectAnswers   []string        `json:"correctAnswers,omitempty"`
}

type OptionInput struct {
	Text string        `json:"text"`
	GoTo *BranchAction `json:"goTo,omitempty"`
}
