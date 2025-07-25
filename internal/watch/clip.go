package watch

import (
	"regexp"
	"strings"
	"time"

	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

// Read user input from clipboard and write response to clipboard
func WatchClipboard(cfg *api.AppConfig) error {
	const trigger = "#"
	const marker = "/*--------*/"
	const interval = 1800 * time.Millisecond

	log.Debugf("WatchClipboard trigger %s...\n", trigger)

	// response content
	isMarker := func(s string) bool {
		return strings.HasPrefix(s, marker)
	}

	isReset := func(s string) bool {
		re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(trigger) + `\s*(?i:(?:reset))\s*`)
		return re.MatchString(s)
	}

	isOn := func(s string) bool {
		re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(trigger) + `\s*(?i:(?:on))\s*`)
		return re.MatchString(s)
	}

	isOff := func(s string) bool {
		re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(trigger) + `\s*(?i:(?:off))\s*`)
		return re.MatchString(s)
	}

	isTodo := func(s string) bool {
		re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(trigger) + `\s*(?i:(?:todo))\s*`)
		return re.MatchString(s)
	}

	readLine := func(s string) string {
		for i := 0; i < len(s); i++ {
			if s[i] == '\n' {
				return s[:i]
			}
		}
		return s
	}

	//
	var watching = true
	clipboard := util.NewClipboard()

	if err := clipboard.Clear(); err != nil {
		return err
	}

	// read until trigger word 'todo' is encountered
	readInput := func() (*api.UserInput, error) {
		var in *api.UserInput
		var pb []string
		for {
			log.Promptf("Watching %v [%v]...\n", watching, len(pb))
			time.Sleep(interval)

			v, err := clipboard.Read()
			if err != nil {
				return nil, err
			}

			line := readLine(v)

			// skip and retain the content if it is a response
			if isMarker(line) {
				continue
			}

			if isReset(line) {
				pb = []string{}
				clipboard.Clear()
				continue
			}

			if isOff(line) {
				watching = false
				clipboard.Clear()
				continue
			}

			if isOn(line) {
				watching = true
				clipboard.Clear()
				continue
			}

			// ignore if not watching
			if !watching {
				continue
			}

			log.Infof("%s\n\n", line)

			// new prompt or content
			clipboard.Clear()

			// embedded request
			if isTodo(line) {
				log.Debugf("found ai command: %s\n", line)

				in, err = parseUserInput(v, trigger)
				if err != nil {
					// treat as regular input
					log.Errorf("Error parsing user input: %s\n", err)
					pb = append(pb, v)
					continue
				}

				if in.Agent == "" {
					in.Agent = cfg.Agent
				}
				if in.Command == "" {
					in.Command = cfg.Command
				}

				log.Debugf("agent: %s %s %v\n", in.Agent, in.Command)

				in.Message = strings.Join(pb, "\n") + in.Message
				break
			}

			// continue appending
			pb = append(pb, v)
		}

		return in, nil
	}

	writeOutput := func(out string) error {
		s := marker + "\n" + out
		if err := clipboard.Write(s); err != nil {
			return err
		}
		return nil
	}

	run := func() {
		in, err := readInput()
		if err != nil {
			log.Errorf("Error reading from clipboard: %s\n", err)
			return
		}

		// run
		cfg.Format = "text"
		if err := agent.RunSwarm(cfg, in); err != nil {
			log.Errorf("Error running agent: %s\n", err)
			util.Alert(err.Error())
			return
		}

		//success
		log.Infof("ai executed successfully\n")
		if err := writeOutput(cfg.Stdout); err != nil {
			log.Debugf("failed to copy content to clipboard: %v\n", err)
			util.Alert(err.Error())
			return
		}

		util.Notify("Done")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		run()
	}
}
