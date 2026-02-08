package md

import (
	"bytes"
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

func collectYAML(start ast.Node, source []byte) (string, ast.Node, error) {
	var b bytes.Buffer
	current := start
	for current != nil {
		nextCur := current.NextSibling()
		switch current := current.(type) {
		case *ast.Heading, *ast.ThematicBreak:
			return b.String(), current, nil
		case *ast.FencedCodeBlock:
			cb := current
			info := ""
			if cb.Info != nil {
				info = strings.TrimSpace(string(cb.Info.Text(source)))
			}
			if info == "yaml" || info == "" {
				b.WriteString(collectCodeContent(cb, source))
			} else {
				return b.String(), current, nil
			}
		default:
			lines := current.Lines()
			if lines != nil {
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					b.Write(line.Value(source))
				}
			}
		}
		current = nextCur
	}
	return b.String(), nil, nil
}

func Parse(source string) (*api.TaskFile, error) {
	// basic sanity checks: unmatched fences or thematic breaks indicate bad files
	if strings.Count(source, "```")%2 == 1 {
		return nil, fmt.Errorf("unterminated fenced code block")
	}
	// count lines with only '---'
	dashRe := regexp.MustCompile(`(?m)^\s*-{3,}\s*$`)
	if len(dashRe.FindAllStringIndex(source, -1))%2 == 1 {
		return nil, fmt.Errorf("unterminated yaml block")
	}

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))
	tf := &api.TaskFile{
		Tasks: make(map[string][]*api.Task),
	}
	// front comment
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
				if currentTask != nil {
					tf.Tasks[currentGroup] = append(tf.Tasks[currentGroup], currentTask)
				}
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
		case *ast.ThematicBreak:
			yStr, skipTo, err := collectYAML(next, sourceBytes)
			if err != nil {
				return nil, err
			}
			yStr = strings.TrimSpace(yStr)
			if yStr != "" {
				if currentTask != nil {
					parseTaskYAML([]byte(yStr), currentTask)
				} else {
					var m map[string]interface{}
					if yaml.Unmarshal([]byte(yStr), &m) == nil {
						bs, _ := yaml.Marshal(m)
						if tf.Arguments == "" {
							tf.Arguments = string(bs)
						} else {
							tf.Arguments += "\n" + string(bs)
						}
					}
				}
			}
			node = skipTo
			continue
		}
		node = next
	}
	if currentTask != nil {
		tf.Tasks[currentGroup] = append(tf.Tasks[currentGroup], currentTask)
	}
	// validation
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
