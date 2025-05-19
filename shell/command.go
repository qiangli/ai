package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	arg "github.com/alexflint/go-arg"
	glob "github.com/bmatcuk/doublestar/v4"
	"github.com/mattn/go-shellwords"

	"github.com/qiangli/ai/internal/log"
	edit "github.com/qiangli/ai/shell/edit"
	fm "github.com/qiangli/ai/shell/explore"
)

func execCommand(shellBin, original string, save bool) error {
	log.Debugf("original command: %q\n", original)

	parsed, err := parseCommand(original)
	if err != nil {
		return err
	}
	log.Debugf("parsed command: %+v\n", parsed)

	// Handle special case for "cd" command
	// assume single argument for "cd" command
	// update PWD environment variable as needed
	if len(parsed) > 0 && len(parsed[0]) > 0 && parsed[0][0] == "cd" {
		if len(parsed[0]) > 1 {
			dir := parsed[0][1]
			dir = subst(dir)
			return Chdir(dir)
		}

		// default to GET_ROOT or HOME if no argument is provided
		// if the the current working directory is a subdirectory of the git, cd to git root
		// otherwise, cd to home directory
		wd, _ := os.Getwd()
		gitRoot := os.Getenv(gitRootEnv)
		if strings.HasPrefix(wd, gitRoot) && len(wd) > len(gitRoot) {
			Chdir(gitRoot)
			return nil
		}

		user, err := user.Current()
		if err != nil {
			return err
		}
		return Chdir(user.HomeDir)
	}

	var modified []string
	for _, part := range parsed {
		if len(part) == 1 && isSep(part[0]) {
			modified = append(modified, part[0])
		} else {
			modified = append(modified, strings.Join(part, " "))
		}
	}

	log.Debugf("modified command: %+v\n", modified)

	//
	cmdline := strings.Join(modified, " ")
	command, page := RemovePageSuffix(cmdline)
	log.Debugf("Executing command: %q\n", command)

	// Execute the command
	capture := wordCompleter.Capture

	// TODO command: set tty=on|off
	ttyOn, _ := strconv.ParseBool(os.Getenv("AI_TTY"))
	if !ttyOn {
		err = RunAndCapture(shellBin, command, page, save, capture)
	} else {
		//TODO: experimental, still buggy
		err = RunPtyCapture(shellBin, command, capture)
		// err = RunNoCapture(shellBin, command)
	}

	// add command as stdin and signal end of processing
	capture(0, command)
	capture(99, "\n")

	return err
}

func RunNoCapture(shellBin, command string) error {
	cmd := exec.Command(shellBin, "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RunAndCapture runs a command and captures its output line by line.
func RunAndCapture(shellBin, command string, page, save bool, capture func(which int, line string) error) error {
	log.Debugf("RunAndCapture: %q page: %v\n", command, page)

	// TTY=1 key=val ... cmd args
	parts := strings.Split(command, " ")
	var ttyOn bool
	for _, part := range parts {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				if k == "AI_TTY" {
					ttyOn, _ = strconv.ParseBool(v)
					break
				}
			}
			continue
		}
		break
	}

	log.Debugf("Running command: %q page: %v tty: %v\n", command, page, ttyOn)

	cmd := exec.Command(shellBin, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if ttyOn {
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	// only capture the output if tty is off
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// a copy of the output verbatim
	var out bytes.Buffer
	go func() {
		reader := bufio.NewReader(stdout)
		var lineBuf []byte
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				data := buf[:n]
				// Write output immediately
				out.Write(data)
				os.Stdout.Write(data)
				// Buffer data for capture
				start := 0
				for i, b := range data {
					if b == '\n' {
						lineBuf = append(lineBuf, data[start:i+1]...)
						_ = capture(1, string(lineBuf))
						lineBuf = lineBuf[:0]
						start = i + 1
					}
				}
				if start < n {
					lineBuf = append(lineBuf, data[start:]...)
				}
			}
			if err != nil {
				break
			}
		}
		// Capture any remaining data as the last line
		if len(lineBuf) > 0 {
			_ = capture(1, string(lineBuf))
		}
	}()

	if err := cmd.Wait(); err != nil {
		return err
	}

	clip := func(s string, max int) string {
		if len(s) > max {
			return s[:max]
		}
		return s
	}

	outText := out.String()
	if save {
		os.Setenv("OUT", clip(outText, 32000))
	}

	// // TODO run more/less if they are requested
	// command | page
	if page {
		return pager(outText)
	}

	return nil
}

// RemovePageSuffix removes a "| page" command suffix.
// Returns the new command and true if a suffix was removed, otherwise returns the original command and false.
func RemovePageSuffix(command string) (string, bool) {
	re := regexp.MustCompile(`(?i)\|\s*(page|more|less)\b\s*(.*)$`)
	matches := re.FindStringSubmatch(command)
	if matches != nil {
		trimmed := strings.TrimSpace(matches[2])
		if trimmed != "" {
			return strings.TrimSpace(command[:matches[0][0]]) + " | " + trimmed, true
		}
		return strings.TrimSpace(command[:len(command)-len(matches[0])]), true
	}
	return command, false
}

var commandSeps = []string{">>", "<<", ">", "<", "&&", "&", "||", "|", ";"}

var isSep = func(s string) bool {
	return slices.Contains(commandSeps, s)
}

func parseCommand(original string) ([][]string, error) {
	// Check if the command needs expansion
	// internal commands need expansion unlike external commands which is taken care of by the shell
	needsExpand := func(s string) bool {
		return slices.Contains([]string{"cd"}, s)
	}

	expand := func(sub string) ([]string, error) {
		args := strings.Fields(sub)

		if len(args) == 0 {
			return args, nil
		}

		// args[0] is the command name
		if needsExpand(args[0]) {
			if err := substAll(args); err != nil {
				return nil, err
			}
		}

		// remove empty arguments
		var filtered []string
		for _, arg := range args {
			arg = strings.TrimSpace(arg)
			if arg != "" {
				filtered = append(filtered, arg)
			}
		}
		return filtered, nil
	}

	// Split the command string by the separators
	// and recursively expand each part.
	// This is a recursive function to handle nested commands.
	var split func(s string, seps []string) []string

	split = func(s string, seps []string) []string {
		log.Debugf("Splitting: %q with separators: %+v\n", s, seps)

		if len(seps) == 0 {
			return []string{s}
		}

		var result []string
		parts := strings.Split(s, seps[0])
		for i, part := range parts {
			if i > 0 {
				result = append(result, seps[0])
			}
			subs := split(part, seps[1:])
			result = append(result, subs...)
		}
		return result
	}

	// subcommands
	var subcommands = split(original, commandSeps)
	var expanded [][]string

	for _, sub := range subcommands {
		if isSep(sub) {
			expanded = append(expanded, []string{sub})
			continue
		}
		c, err := expand(sub)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, c)
	}
	return expanded, nil
}

func substAll(args []string) error {
	for i, arg := range args {
		p, err := expandPath(arg)
		if err != nil {
			return err
		}
		if len(p) > 0 {
			args[i] = strings.Join(p, string(os.PathListSeparator))
		}
	}
	return nil
}

func expandPath(s string) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	var p = s

	// expand environment variables
	if strings.Contains(p, "$") {
		p = os.ExpandEnv(p)
	}
	// expand tilde (~) for home directory
	if strings.HasPrefix(p, "~") {
		p = strings.Replace(p, "~", home, 1)
	}

	// globbing
	// golang filepath.Glob(p) not working for some patterns supported by shell
	return glob.FilepathGlob(p)
}

func subst(s string) string {
	p, err := expandPath(s)
	if err != nil || len(p) == 0 {
		return s
	}
	return strings.Join(p, string(os.PathListSeparator))
}

func runHistory(args string) error {
	if args == "-c" {
		clearHistory()
	} else {
		showHistory()
	}
	return nil
}

func runAlias(args string) error {
	if args == "" {
		showAlias()
		return nil
	}

	parts := strings.SplitN(args, "=", 2)
	if len(parts) == 1 {
		k := strings.TrimSpace(parts[0])
		fmt.Println("  \033[0;32m" + k + " \033[0m =  " + aliasRegistry[k])
	} else {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if v != "" {
			v = subst(v)
		}
		aliasRegistry[k] = v
	}
	return nil
}

func showAlias() {
	var keys []string
	for k := range aliasRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	width := 0
	for _, k := range keys {
		if len(k) > width {
			width = len(k)
		}
	}

	for _, k := range keys {
		padded := fmt.Sprintf("%-*s", width, k)
		fmt.Println("  \033[0;32m" + padded + " \033[0m =  " + aliasRegistry[k])
	}
}

func runEnv(args string) error {
	if args == "" {
		showEnv()
		return nil
	}

	parts := strings.SplitN(args, "=", 2)
	if len(parts) == 1 {
		k := strings.TrimSpace(parts[0])
		v := os.Getenv(k)
		fmt.Println("  \033[0;32m" + k + "\033[0;33m=\033[0m" + v)
	} else {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if v == "" {
			os.Unsetenv(k)
		} else {
			// special handling for PATH
			// treat each path as a separate entry
			if k == "PATH" {
				paths := strings.Split(v, string(os.PathListSeparator))
				if err := substAll(paths); err != nil {
					return err
				}
				v = strings.Join(paths, string(os.PathListSeparator))
			} else {
				v = subst(v)
			}
			os.Setenv(k, v)
		}
	}
	return nil
}

func showEnv() {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	var keys []string
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Println("  \033[0;32m" + k + "\033[0;33m=\033[0m" + envMap[k])
	}
}

var exploreConfig = &fm.Config{
	// OpenWith:      "txt:less -N;go:vim;md:glow -p",
	OpenWith:      "",
	MainColor:     "#0000FF",
	WithHighlight: true,
	StatusBar:     "Mode() + ' ' + Size() + ' ' + ModTime()",
	ShowIcons:     false, // fNerd Fonts required
	DirOnly:       false,
	Preview:       true,
	HideHidden:    false,
	WithBorder:    true,
	Fuzzy:         true,
	SortBy:        fm.SortByName,
	Reverse:       false,
}

func runExplore(s string) error {
	var opts struct {
		Path    string `arg:"positional"`
		Chdir   bool   `arg:"-C,--cd" help:"chdir to the last visited path"`
		SortBy  string `arg:"-s,--sort" help:"sort by name, time, or size"`
		Reverse bool   `arg:"-r,--reverse" help:"reverse the order of the sort"`
		All     bool   `arg:"-a,--all" help:"include directory entries whose names begin with a dot (.)"`
		Help    bool   `arg:"-h,--help" help:"show help"`
	}

	usage := func() {
		fmt.Println("Usage: explore [options] [path]")
		fmt.Println("Options:")
		fmt.Println("  -C, --cd       chdir to the last visited path")
		fmt.Println("  -s, --sort     sort by name, time, or size")
		fmt.Println("  -r, --reverse  reverse the order of the sort")
		fmt.Println("  -h, --help     show help")
		fmt.Println("")
		showExploreKeyBindings()
		fmt.Println("")
	}

	parser, err := arg.NewParser(arg.Config{}, &opts)
	if err != nil {
		return err
	}

	args, err := shellwords.Parse(s)
	if err != nil {
		return err
	}

	err = parser.Parse(args)
	switch {
	case err == arg.ErrHelp:
		usage()
		return nil
	case err != nil:
		return err
	}

	switch opts.SortBy {
	case "name":
		exploreConfig.SortBy = fm.SortByName
	case "time":
		exploreConfig.SortBy = fm.SortByModTime
	case "size":
		exploreConfig.SortBy = fm.SortBySize
	default:
		if opts.SortBy != "" {
			usage()
			return nil
		}
	}

	//
	p := opts.Path

	if p != "" {
		if ex, err := expandPath(p); err == nil && len(ex) > 0 {
			p = ex[0]
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("path %q not found: %w", p, err)
		}
	} else {
		var err error
		p, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	exploreConfig.Roots = []string{p}
	exploreConfig.HideHidden = !opts.All
	visited, err := fm.Explore(exploreConfig)
	if err != nil {
		return err
	}
	if opts.Chdir {
		if err := Chdir(visited); err != nil {
			return err
		}
	} else {
		visitedRegistry.Visit(visited)
	}
	return nil
}

func runEdit(s string) error {
	var opts struct {
		File []string `arg:"positional"`
		Help bool     `arg:"-h,--help" help:"show help"`
	}

	usage := func() {
		fmt.Println("Usage: edit [file]")
		fmt.Println("Options:")
		fmt.Println("  -h, --help     show help")
		fmt.Println("")
		showEditKeyBindings()
		fmt.Println("")
	}

	parser, err := arg.NewParser(arg.Config{}, &opts)
	if err != nil {
		return err
	}

	args, err := shellwords.Parse(s)
	if err != nil {
		return err
	}

	err = parser.Parse(args)
	switch {
	case err == arg.ErrHelp:
		usage()
		return nil
	case err != nil:
		return err
	}

	return edit.Edit(opts.File)
}
