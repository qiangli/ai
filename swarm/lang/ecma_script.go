package lang

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
)

func Javascript(ctx context.Context, script string) (any, error) {
	vm := goja.New()
	v, err := vm.RunString(script)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("%v", v), nil
}
