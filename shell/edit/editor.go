package edit

import (
	"fmt"

	"github.com/zyedidia/micro/v2/editor"
)

// https://github.com/qiangli/micro.git
// https://github.com/zyedidia/micro.git
// Usage: micro [OPTIONS] [FILE]...
func Edit(args []string) error {
	go editor.NewEditor(args)

	rc := <-editor.Exiting
	if rc != 0 {
		return fmt.Errorf("exit code: %d", rc)
	}
	return nil
}
