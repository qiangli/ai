package api

// Task represents a Task.
// Level 2 - 6 heading
// https://spec.commonmark.org/0.31.2/#atx-headings
// https://spec.commonmark.org/0.31.2/#setext-headings
type Task struct {
	// 2 - 6 heading before the separators of one or more dash/"-"
	// name consists of ^[a-z0-9_]+$ after transformation.
	// + all charaters not in the set after converted to underscore/"_"
	// + all upper case letters are converted into their lower case form.
	Name string `json:"name"`
	// 2 - 6 heading after the separators
	Display string `json:"display"`
	// first paragraph under the heading
	Description string `json:"description"`

	// https://spec.commonmark.org/0.31.2/#fenced-code-block
	// ```mime_type
	// [content]
	// ```
	MimeType string `json:"mime_type"`
	Content  string `json:"content"`

	// map of name:value extracted after the text: 'arguments:' in yaml style
	// separated by optional [thematic-breaks](https://spec.commonmark.org/0.31.2/#thematic-breaks)
	// Example:
	// ---
	// arguments:
	//   log_level: info
	//   max_turns: 50
	// ---
	Arguments map[string]any `json:"arguments"`

	// list of task names after the text: 'dependencies:' in yaml style
	// separated by optional [thematic-breaks](https://spec.commonmark.org/0.31.2/#thematic-breaks)
	// Examples:
	// ---
	// dependencies:
	//   - build
	//   - test
	//   - install
	// ---
	Dependencies []*Task `json:"dependencies"`
}

// https://commonmark.org/
// https://spec.commonmark.org/0.31.2/
type TaskFile struct {
	// Level 1 heading
	// https://spec.commonmark.org/0.31.2/#atx-headings
	// https://spec.commonmark.org/0.31.2/#setext-headings
	Title string `json:"title"`

	// First paragraph under the heading
	// https://spec.commonmark.org/0.31.2/#paragraphs
	Description string `json:"description"`

	// Text extractd from first comment line of the file in command line option/flag style
	// https://spec.commonmark.org/0.31.2/#html-comment
	// Examples
	// <!-- /usr/bin/env ai /sh:run_task --task-name default --script -->
	Arguments string `json:"arguments"`

	// Level 2 - 6 heading
	Tasks map[string][]*Task
}
