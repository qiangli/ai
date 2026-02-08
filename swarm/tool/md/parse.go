package md

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/yuin/goldmark"
	ast "github.com/yuin/goldmark/ast"
	text "github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

func normalizeName(s string) string {
	s = strings.ToLower(s)
	r := regexp.MustCompile(`[^a-z0-9]+`)
	s = r.ReplaceAllString(s, "-")
	r2 := regexp.MustCompile(`-+`)
	s = r2.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unnamed"
	}
	return s
}

func nodeText(n ast.Node, source []byte) string {
	// Node.Text returns a byte slice for the node's text content.
	return strings.TrimSpace(string(n.Text(source)))
}

func collectCodeContent(cb *ast.FencedCodeBlock, source []byte) string {
	var sb strings.Builder
	lines := cb.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		sb.Write(line.Value(source))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func htmlBlockText(block *ast.HTMLBlock, source []byte) string {
	var b strings.Builder
	lines := block.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	if block.HasClosure() {
		cl := block.ClosureLine
		b.Write(cl.Value(source))
	}
	return b.String()
}

func parseTaskYAML(y []byte, task *api.Task) error {
	var m map[string]interface{}
	if err := yaml.Unmarshal(y, &m); err != nil {
		return err
	}

	if argI, ok := m["arguments"]; ok {
		if args, ok := argI.(map[interface{}]interface{}); ok {
			taskArg := map[string]any{}
			for k, v := range args {
				if ks, ok := k.(string); ok {
					taskArg[ks] = v
				}
			}
			task.Arguments = taskArg
		} else if args, ok := argI.(map[string]interface{}); ok {
			task.Arguments = args
		}
	}
	if depI, ok := m["dependencies"]; ok {
		var deps []string
		if dv, ok := depI.([]interface{}); ok {
			for _, di := range dv {
				if ds, ok := di.(string); ok {
					deps = append(deps, ds)
				}
			}
		} else if dv, ok := depI.([]string); ok {
			deps = dv
		}
		for _, ds := range deps {
			dname := normalizeName(ds)
			task.Dependencies = append(task.Dependencies, &api.Task{Name: dname, Display: ds})
		}
	}
	return nil
}

// findThematicBreakPairs finds pairs of '---' thematic breaks in the source
// Returns a list of [start, end] index pairs where:
// - start is the end of the opening '---' line (after the newline)
// - end is the start of the closing '---' line
func findThematicBreakPairs(source string) [][2]int {
	// Find all lines that are thematic breaks (3+ dashes, optional surrounding whitespace)
	dashRe := regexp.MustCompile(`(?m)^[ \t]*(-{3,})[ \t]*$`)
	allMatches := dashRe.FindAllStringIndex(source, -1)

	var pairs [][2]int
	for i := 0; i+1 < len(allMatches); i += 2 {
		// Start position: end of first --- line (including newline if present)
		start := allMatches[i][1]
		if start < len(source) && source[start] == '\n' {
			start++
		}
		// End position: start of second --- line
		end := allMatches[i+1][0]
		pairs = append(pairs, [2]int{start, end})
	}
	return pairs
}

// collectRawFromNode collects the original source bytes for a node by reading its
// Lines() segments if available, otherwise recursively collecting from children.
func collectRawFromNode(n ast.Node, source []byte) string {
	if n == nil {
		return ""
	}
	if lines := n.Lines(); lines != nil && lines.Len() > 0 {
		var sb strings.Builder
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			sb.Write(line.Value(source))
			// Preserve original line breaks between segments
			sb.WriteByte('\n')
		}
		return sb.String()
	}
	// fallback: recurse into children
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		sb.WriteString(collectRawFromNode(c, source))
	}
	return sb.String()
}

// collectYAMLBetweenBreaks collects the raw text between an opening thematic break node
// and the next closing thematic break in the AST. It returns the YAML text (trimmed)
// and the AST node following the closing thematic break (may be nil).
func collectYAMLBetweenBreaks(start ast.Node, source []byte) (string, ast.Node) {
	if start == nil {
		return "", nil
	}
	var sb strings.Builder
	for node := start.NextSibling(); node != nil; node = node.NextSibling() {
		if _, isBreak := node.(*ast.ThematicBreak); isBreak {
			return strings.TrimSpace(sb.String()), node.NextSibling()
		}
		// Collect raw source for the node to preserve YAML formatting
		switch n := node.(type) {
		case *ast.List:
			// Reconstruct list items with hyphen markers
			for li := n.FirstChild(); li != nil; li = li.NextSibling() {
				raw := strings.TrimSpace(collectRawFromNode(li, source))
				if raw == "" {
					continue
				}
				lines := strings.Split(raw, "\n")
				for _, l := range lines {
					if strings.TrimSpace(l) == "" {
						continue
					}
					sb.WriteString("- ")
					sb.WriteString(strings.TrimSpace(l))
					sb.WriteString("\n")
				}
			}
		default:
			raw := collectRawFromNode(node, source)
			if strings.TrimSpace(raw) != "" {
				sb.WriteString(raw)
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// Exported wrapper in case other packages expect the exported symbol
func CollectYAMLBetweenBreaks(start ast.Node, source []byte) (string, ast.Node) {
	return collectYAMLBetweenBreaks(start, source)
}

func Parse(source string) (*api.TaskFile, error) {
	// basic sanity checks
	if strings.Count(source, "```")%2 == 1 {
		return nil, fmt.Errorf("unterminated fenced code block")
	}

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	// Find all thematic break pairs for YAML blocks
	yamlPairs := findThematicBreakPairs(source)

	// Count total thematic breaks to detect unpaired ones
	dashRe := regexp.MustCompile(`(?m)^[ \t]*(-{3,})[ \t]*$`)
	totalBreaks := len(dashRe.FindAllStringIndex(source, -1))
	if totalBreaks%2 == 1 {
		return nil, fmt.Errorf("unterminated YAML block (unpaired ---)")
	}

	tf := &api.TaskFile{
		Tasks: make(map[string][]*api.Task),
	}

	// Extract front comment for Arguments
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if block, ok := n.(*ast.HTMLBlock); ok {
				s := strings.TrimSpace(htmlBlockText(block, sourceBytes))
				if strings.HasPrefix(s, "<!--") && strings.HasSuffix(s, "-->") {
					tf.Arguments = strings.TrimSpace(s[4 : len(s)-3])
					return ast.WalkStop, nil
				}
			}
		}
		return ast.WalkContinue, nil
	})

	currentGroup := "default"
	var currentTask *api.Task
	titleSet := false
	processedYAMLPairs := make(map[int]bool) // Track which YAML pairs we've processed

	node := doc.FirstChild()
	for node != nil {
		next := node.NextSibling()

		switch v := node.(type) {
		case *ast.Heading:
			level := int(v.Level)
			d := nodeText(v, sourceBytes)
			name := normalizeName(d)
			if level == 1 {
				tf.Title = d
				titleSet = true
			} else if level >= 3 {
				// Save previous task
				if currentTask != nil {
					tf.Tasks[currentGroup] = append(tf.Tasks[currentGroup], currentTask)
				}
				// Create new task
				currentTask = &api.Task{
					Name:    name,
					Display: d,
				}
			}

		case *ast.Paragraph:
			if titleSet && tf.Description == "" {
				tf.Description = nodeText(v, sourceBytes)
			}
			if currentTask != nil && currentTask.Description == "" {
				currentTask.Description = nodeText(v, sourceBytes)
			}

		case *ast.ThematicBreak:
			// Check if this is the start of a YAML block
			// We need to find which YAML pair this belongs to
			// Look at the next sibling - if it's between a pair of breaks, this is the opening break
			if next != nil {
				// Find a YAML pair that starts around this position
				for pairIdx, pair := range yamlPairs {
					if processedYAMLPairs[pairIdx] {
						continue
					}
					// Check if the content between the pair contains nodes between current and some future node
					yamlText := strings.TrimSpace(source[pair[0]:pair[1]])
					if yamlText != "" {
						// Check if this looks like it's the right YAML block by seeing if the next content matches
						// For now, just try to apply it and mark as processed
						if currentTask != nil {
							if err := parseTaskYAML([]byte(yamlText), currentTask); err == nil {
								processedYAMLPairs[pairIdx] = true
								// Skip past the YAML block in the AST
								// Find the next thematic break and skip past it
								for next != nil {
									if _, isBreak := next.(*ast.ThematicBreak); isBreak {
										next = next.NextSibling()
										break
									}
									next = next.NextSibling()
								}
								break
							}
						} else {
							// Global YAML
							var m map[string]interface{}
							if yaml.Unmarshal([]byte(yamlText), &m) == nil {
								bs, _ := yaml.Marshal(m)
								if tf.Arguments == "" {
									tf.Arguments = string(bs)
								} else {
									tf.Arguments += "\n" + string(bs)
								}
								processedYAMLPairs[pairIdx] = true
								// Skip past the YAML block
								for next != nil {
									if _, isBreak := next.(*ast.ThematicBreak); isBreak {
										next = next.NextSibling()
										break
									}
									next = next.NextSibling()
								}
								break
							}
						}
					}
				}
			}

		case *ast.FencedCodeBlock:
			if currentTask != nil && currentTask.MimeType == "" {
				lang := v.Language(sourceBytes)
				if len(lang) > 0 {
					currentTask.MimeType = strings.TrimSpace(string(lang))
				} else {
					currentTask.MimeType = "text"
				}
				currentTask.Content = collectCodeContent(v, sourceBytes)
			}
		}

		node = next
	}

	// Save last task
	if currentTask != nil {
		tf.Tasks[currentGroup] = append(tf.Tasks[currentGroup], currentTask)
	}

	// Validation
	for g, tasks := range tf.Tasks {
		for idx, t := range tasks {
			if strings.TrimSpace(t.Display) == "" {
				return nil, fmt.Errorf("task missing display in group '%s' index %d", g, idx)
			}
			matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, t.Name)
			if !matched {
				return nil, fmt.Errorf("invalid task name '%s' for display '%s' in group '%s'", t.Name, t.Display, g)
			}
		}
	}
	if tf.Title == "" && tf.Arguments == "" && len(tf.Tasks) == 0 {
		return nil, errors.New("no content parsed")
	}
	return tf, nil
}

func Dump(tf *api.TaskFile) {
	fmt.Printf("Title: %s\n", tf.Title)
	fmt.Printf("Arguments: %s\n", tf.Arguments)
	fmt.Printf("Description: %s\n", tf.Description)
	for k, v := range tf.Tasks {
		fmt.Printf("Group: %s (count=%d)\n", k, len(v))
		for _, t := range v {
			fmt.Printf(" - Task: %s display=%s mime=%s desc=%s deps=%d\n", t.Name, t.Display, t.MimeType, t.Description, len(t.Dependencies))
		}
	}
}
