//go:build windows

package explore

func (e Env) Owner() (string, error) {
	return "¯\\_(ツ)_/¯", nil
}
