# PR Review Guide

## Estimated Effort to Review [1-5]
- **Estimate:**: {{ .Review.EstimatedEffortToReview }}

## Quality [0-100]
- **Score**: {{ .Review.Score }}

## Relevant Tests
- **{{ .Review.RelevantTests }}**

## Insights from User Answers
- {{ .Review.InsightsFromUserAnswers }}

## Key Issues to Review
{{ range .Review.KeyIssuesToReview }}
- **File**: {{ .RelevantFile }}
  - **Header**: {{ .IssueHeader }}
  - **Content**: {{ .IssueContent }}
  - **Lines**: {{ .StartLine }}-{{ .EndLine }}
{{ end }}

## Security Concerns
- {{ .Review.SecurityConcerns }}