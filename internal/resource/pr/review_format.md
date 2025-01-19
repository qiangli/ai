# PR Review Guide

## Estimated Effort to Review [1-5]
- **Estimate:**: {{ .EstimatedEffortToReview }}

## Quality [0-100]
- **Score**: {{ .Score }}

## Relevant Tests
- **{{ .RelevantTests }}**

## Insights from User Answers
- {{ .InsightsFromUserAnswers }}

## Key Issues to Review
{{ range .KeyIssuesToReview }}
- **File**: {{ .RelevantFile }}
  - **Header**: {{ .IssueHeader }}
  - **Content**: {{ .IssueContent }}
  - **Lines**: {{ .StartLine }}-{{ .EndLine }}
{{ end }}

## Security Concerns
- {{ .SecurityConcerns }}