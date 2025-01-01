package internal

import (
	"fmt"
	"os"

	"github.com/qiangli/ai/internal/log"
)

// UserInputError represents user input error.
type UserInputError struct {
	text string
}

func (r *UserInputError) Error() string {
	return r.text
}

func NewUserInputError(text string) error {
	return &UserInputError{
		text: text,
	}
}

func NewUserInputErrorf(format string, a ...interface{}) error {
	return &UserInputError{
		text: fmt.Sprintf(format, a...),
	}
}

// Exit checks error and exit with the following code:
// 0 -- no error
// 1 -- general failure
// 2 -- user error
func Exit(err error) {
	if err == nil {
		os.Exit(0)
	}

	const max = 500
	errMsg := err.Error()
	if !log.IsVerbose() && len(errMsg) > max {
		errMsg = errMsg[:max] + "..."
	}

	log.Errorln(errMsg)

	switch err.(type) {
	case *UserInputError:
		os.Exit(2)
	default:
		os.Exit(1)
	}
}
