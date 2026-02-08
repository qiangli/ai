package atm

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/qiangli/ai/swarm/api"
)

func clip(s string, max int) string {
	if max > 0 && len(s) > max {
		trailing := "..."
		if s[len(s)-1] == '\n' || s[len(s)-1] == '\r' {
			trailing = "...\n"
		}
		s = s[:max] + trailing
	}
	return s
}

func toPascalCase(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	tc := cases.Title(language.English)

	for i := range words {
		words[i] = tc.String(words[i])
	}
	return strings.Join(words, "")
}

// // baseCommand returns the last part of the string separated by /.
// func baseCommand(s string) string {
// 	s = strings.TrimSpace(s)
// 	s = strings.Trim(s, "/")
// 	sa := strings.Split(s, "/")
// 	return sa[len(sa)-1]
// }

// split2 splits string s by sep into two parts. if there is only one part,
// use val as the second part
func split2(s string, sep string, val string) (string, string) {
	var p1, p2 string
	parts := strings.SplitN(s, sep, 2)
	if len(parts) == 1 {
		p1 = parts[0]
		p2 = val
	} else {
		p1 = parts[0]
		p2 = parts[1]
	}
	return p1, p2
}

func nvl(sa ...string) string {
	for _, v := range sa {
		if v != "" {
			return v
		}
	}
	return ""
}

func PrettyJSON(obj any) (string, error) {
	v, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return "", fmt.Errorf("Error marshaling JSON: %v", err)
	}
	return string(v), nil
}

// for mock tests
// respect --option adapter=echo
func echoAdapter(args api.ArgMap) (any, error) {
	if len(args) == 0 {
		return "", nil
	}

	// custom response: echo__id
	id := args.Kitname().ID()
	if v, found := args["echo__"+id]; found {
		return api.ToString(v), nil
	}

	v, err := json.Marshal(args)
	return string(v), err
}

func parseCmdline(s string) string {
	if s == "" {
		return ""
	}
	sa := strings.SplitN(s, "\n", 2)
	cmdline := sa[0]
	if after, ok := strings.CutPrefix(cmdline, "#!"); ok {
		cmdline = after
	}
	sa = strings.SplitN(cmdline, " ", 2)
	// /agent:pack/sub
	// /kit:name
	if strings.Contains(sa[0], ":") {
		return cmdline
	}
	// expected:
	// ai [action] or [action]
	return sa[1]
}
