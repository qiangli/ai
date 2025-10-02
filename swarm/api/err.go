package api

type UnsupportedError string

func (s UnsupportedError) Error() string {
	return "unsupported feature: " + string(s)
}
