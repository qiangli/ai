package atm

import (
	"slices"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

var defaultServeTriggers = []string{"ai", "#ai", "//ai"}

func isServeTrigger(tok string, trigger string) bool {
	tok = strings.TrimSpace(tok)
	tok = strings.ToLower(tok)
	if tok == "" {
		return false
	}
	if trigger != "" {
		return tok == strings.ToLower(strings.TrimSpace(trigger))
	}
	return slices.Contains(defaultServeTriggers, tok)
}

// serveParseTriggeredFirstLine:
// - ignores leading blanks when considering trigger
// - if trigger matches, removes it and parses the remaining first line as an action command using conf.ParseActionArgs
// - action defaults to "default" if not detected
// - returns (action, payload, ok)
func serveParseTriggeredFirstLine(in string, trigger string) (string, string, bool) {
	in = strings.ReplaceAll(in, "\r\n", "\n")
	lines := strings.Split(in, "\n")
	if len(lines) == 0 {
		return "", "", false
	}
	first := strings.TrimLeft(lines[0], " \t")
	argv := shlex.Argv(first)
	if len(argv) == 0 {
		return "", "", false
	}
	if !isServeTrigger(argv[0], trigger) {
		return "", "", false
	}
	argv = argv[1:]
	if len(argv) == 0 {
		return "default", strings.Join(lines[1:], "\n"), true
	}

	argm0, err := conf.ParseActionArgs(argv)
	if err != nil {
		// treat parse errors as invalid trigger input; keep safe in clipboard/file mode
		return "", "", false
	}
	argm := api.ArgMap(argm0)
	act := argm.Kitname().ID()
	if strings.TrimSpace(act) == "" {
		act = "default"
	}
	payload := strings.Join(lines[1:], "\n")
	payload = strings.TrimLeft(payload, " \t")
	return act, payload, true
}
