# PR Code Suggestions

{{range .CodeSuggestions}}
## Suggestion for Improvement

**Relevant File:**  
`{{.RelevantFile}}`

**Programming Language:**  
`{{.Language}}`

**Suggestion Content:**  
{{.SuggestionContent}}

**Existing Code Snippet:**  
```{{.Language}}
{{.ExistingCode}}
```

**Improved Code Snippet:**  
```{{.Language}}
{{.ImprovedCode}}
```

**Summary:**  
{{.OneSentenceSummary}}

**Label:**  
`{{.Label}}`

---
{{end}}
