package pr

type Type string

type FileFileDescription struct {
	Filename string `json:"filename"`
	Summary  string `json:"changes_summary"`
	Title    string `json:"changes_title"`
	Label    string `json:"label"`
}

type PRDescription struct {
	Types       []Type `json:"type"`
	Description string `json:"description"`
	Title       string `json:"title"`

	Files []*FileFileDescription `json:"pr_files"`
}
type CodeSuggestion struct {
	RelevantFile       string `json:"relevant_file"`
	Language           string `json:"language"`
	SuggestionContent  string `json:"suggestion_content"`
	ExistingCode       string `json:"existing_code"`
	ImprovedCode       string `json:"improved_code"`
	OneSentenceSummary string `json:"one_sentence_summary"`
	Label              string `json:"label"`
}

type PRCodeSuggestions struct {
	CodeSuggestions []CodeSuggestion `json:"code_suggestions"`
}

type KeyIssuesComponentLink struct {
	RelevantFile string `json:"relevant_file"`
	IssueHeader  string `json:"issue_header"`
	IssueContent string `json:"issue_content"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
}

type Review struct {
	EstimatedEffortToReview int                      `json:"estimated_effort_to_review"`
	Score                   string                   `json:"score"`
	RelevantTests           string                   `json:"relevant_tests"`
	InsightsFromUserAnswers string                   `json:"insights_from_user_answers"`
	KeyIssuesToReview       []KeyIssuesComponentLink `json:"key_issues_to_review"`
	SecurityConcerns        string                   `json:"security_concerns"`
}

type PRReview struct {
	Review Review `json:"review"`
}
type Input struct {
	Instruction string
	Diff        string

	ChangeLog string
}
