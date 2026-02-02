package atm

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"

	"github.com/qiangli/ai/swarm/api"
)

// ServeLoop implements an infinite-loop interactive runner.
// It reads input (clipboard/stdin/file), runs an AI action, and writes output (clipboard/stdout/file).
//
// Parameters:
//   - format: raw (default) or markdown
//   - input: clipboard (default), stdin, or file:/path or /abs/path
//   - output: list of destinations: clipboard, stdout, file:/path (default: [clipboard, stdout])
//   - background: if true, detach via nohup and keep running after terminal exits
//   - action: action to execute per request (default: agent:root/root)
//   - poll: clipboard polling interval (default 500ms)
//   - dedupe: skip when input unchanged (default true)
func (r *SystemKit) ServeLoop(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (string, error) {
	// background mode: re-exec self with a private env var, detach via nohup.
	background, _ := api.GetBoolProp("background", args)
	if background {
		// prevent recursion
		if os.Getenv("AI_SERVE_BACKGROUND") != "1" {
			// build a command that calls this same action with background disabled
			// NOTE: we delegate to /bin/bash -lc with nohup.
			format, _ := api.GetStrProp("format", args)
			input, _ := api.GetStrProp("input", args)
			outputs, _ := api.GetArrayProp("output", args)
			action, _ := api.GetStrProp("action", args)
			poll, _ := api.GetStrProp("poll", args)
			dedupe, _ := api.GetBoolProp("dedupe", args)

			// defaults handled in foreground run as well; keep flags explicit here.
			cmd := buildServeReexecCommand(format, input, outputs, action, poll, dedupe)
			nohup := fmt.Sprintf("AI_SERVE_BACKGROUND=1 nohup %s >/dev/null 2>&1 &", shellQuote(cmd))
			_, err := ExecCommand(ctx, vars.OS, vars, "/bin/bash -lc "+shellQuote(nohup), nil)
			if err != nil {
				return "", err
			}
			return "serve started in background", nil
		}
		// fallthrough in background child: keep running, handle signals.
	}

	// cancellation via signals
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		cancel()
	}()

	format, _ := api.GetStrProp("format", args)
	if format == "" {
		format = "raw"
	}
	input, _ := api.GetStrProp("input", args)
	if input == "" {
		input = "clipboard"
	}
	outputs, _ := api.GetArrayProp("output", args)
	if len(outputs) == 0 {
		outputs = []string{"clipboard", "stdout"}
	}
	pollStr, _ := api.GetStrProp("poll", args)
	poll := 1000 * time.Millisecond
	if pollStr != "" {
		if d, err := time.ParseDuration(pollStr); err == nil {
			poll = d
		}
	}
	dedupe := true
	if v, err := api.GetBoolProp("dedupe", args); err == nil {
		dedupe = v
	}

	action, _ := api.GetStrProp("action", args)
	action = strings.TrimSpace(action)

	trigger, _ := api.GetStrProp("trigger", args)
	trigger = strings.TrimSpace(trigger)

	var lastIn string
	for {
		select {
		case <-ctx.Done():
			return "serve loop exited", nil
		default:
		}

		in, err := serveReadInput(ctx, vars, input, poll)
		if err != nil {
			// non-fatal: write error to stdout/clipboard
			_ = serveWriteOutputs(vars, outputs, "ai> "+err.Error()+"\n")
			// avoid tight loop
			time.Sleep(poll)
			continue
		}
		if strings.TrimSpace(in) == "" {
			time.Sleep(poll)
			continue
		}
		if dedupe && in == lastIn {
			time.Sleep(poll)
			continue
		}
		lastIn = in

		// trigger + action parsing from first line (clipboard/file mode requires trigger)
		var payload any
		payload = in
		if input == "stdin" {
			// stdin is interactive; prompt is already "ai> ", trigger is optional
			// action remains as configured (default if empty)
			_ = serveWriteOutputs(vars, outputs, "ai> ")
			_ = serveWriteOutputs(vars, outputs, in+"\n")
		} else {
			var ok bool
			payload, ok = serveParseTriggeredFirstLine(in, trigger)
			if !ok {
				// no trigger word, input ignored
				// print a dot to acknowlege receiving the new input
				if in != lastIn {
					_ = serveWriteOutputs(vars, outputs, ".")
				}
				time.Sleep(poll)
				continue
			}
		}

		out, err := serveRunAction(ctx, vars, action, format, payload)
		if err != nil {
			out = fmt.Sprintf("%v\n", err)
		}

		// always prefix output prompt when writing to stdout (and others, per spec for stdin/stdout)
		outWithPrompt := out
		if input == "stdin" {
			outWithPrompt = "ai> " + out
		}

		if err := serveWriteOutputs(vars, outputs, outWithPrompt); err != nil {
			// best effort
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
		}

		// in clipboard mode, small sleep to avoid busy loop
		if input == "clipboard" {
			time.Sleep(poll)
		}
	}
}

func serveReadInput(ctx context.Context, vars *api.Vars, input string, poll time.Duration) (string, error) {
	switch {
	case input == "stdin":
		// prompt for input
		fmt.Fprint(os.Stdout, "ai> ")
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	case input == "clipboard":
		// poll clipboard; return current text
		v, err := clipboard.ReadAll()
		if err != nil {
			return "", err
		}
		return v, nil
	case strings.HasPrefix(input, "file:"):
		p := strings.TrimPrefix(input, "file:")
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return string(b), nil
	case strings.HasPrefix(input, "/"):
		b, err := os.ReadFile(input)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported input: %s", input)
	}
}

func serveRunAction(ctx context.Context, vars *api.Vars, action, format string, in any) (string, error) {
	// Build args to run the underlying action.
	// We pass query/message depending on kit.
	var args api.ArgMap

	if s, ok := in.(string); ok {
		args = api.ArgMap{}
		args["message"] = s
	} else {
		args = in.(api.ArgMap)
	}

	// override
	if action != "" {
		kit, name := api.Kitname(action).Decode()
		if kit == "agent" {
			pack, sub := api.Packname(name).Decode()
			args["kit"] = "agent"
			args["pack"] = pack
			args["name"] = sub
		} else {
			args["kit"] = kit
			args["name"] = name
			args["pack"] = ""
		}
	}
	if format != "" {
		args["format"] = format
	}

	id := args.Kitname().ID()
	if id == "" {
		// default @root/root
		args["kit"] = "agent"
		args["pack"] = "root"
		args["name"] = "root"
	}

	if msg, ok := args["message"]; ok {
		vars.Global.Set("message", msg)
	}

	res, err := api.Exec(ctx, vars.RootAgent.Runner, args)
	if err != nil {
		return "", err
	}
	return api.ToString(res), nil
}

func serveWriteOutputs(_ *api.Vars, outputs []string, out string) error {
	for _, dst := range outputs {
		dst = strings.TrimSpace(dst)
		switch {
		case dst == "stdout":
			fmt.Fprint(os.Stdout, out)
		case dst == "clipboard":
			if err := clipboard.WriteAll(out); err != nil {
				return err
			}
		case strings.HasPrefix(dst, "file:"):
			p := strings.TrimPrefix(dst, "file:")
			if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported output: %s", dst)
		}
	}
	return nil
}

func buildServeReexecCommand(format, input string, outputs []string, action, poll string, dedupe bool) string {
	var b strings.Builder
	// call the ai binary (assumed in PATH) with sh:serve
	b.WriteString("ai /sh:serve")
	if format != "" {
		b.WriteString(" --format ")
		b.WriteString(shellQuote(format))
	}
	if input != "" {
		b.WriteString(" --input ")
		b.WriteString(shellQuote(input))
	}
	// background disabled in child
	b.WriteString(" --background=false")
	if action != "" {
		b.WriteString(" --action ")
		b.WriteString(shellQuote(action))
	}
	if poll != "" {
		b.WriteString(" --poll ")
		b.WriteString(shellQuote(poll))
	}
	// dedupe
	b.WriteString(" --dedupe=")
	if dedupe {
		b.WriteString("true")
	} else {
		b.WriteString("false")
	}
	for _, o := range outputs {
		b.WriteString(" --output ")
		b.WriteString(shellQuote(o))
	}
	return b.String()
}

func shellQuote(s string) string {
	// minimal single-quote escaping for bash -lc
	if s == "" {
		return "''"
	}
	s = strings.ReplaceAll(s, "'", "'\\''")
	return "'" + s + "'"
}
