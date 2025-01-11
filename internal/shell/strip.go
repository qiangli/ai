package shell

// https://github.com/MatusOllah/stripansi/blob/main/stripansi.go
// https://github.com/acarl005/stripansi/blob/master/stripansi.go
import (
	"io"
	"regexp"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re *regexp.Regexp = regexp.MustCompile(ansi)

// Regexp returns a copy of the underlying [regexp.Regexp].
func Regexp() *regexp.Regexp {
	return re.Copy()
}

// Bytes removes ANSI escape sequences from the byte slice.
func Bytes(b []byte) []byte {
	return re.ReplaceAll(b, nil)
}

// String removes ANSI escape sequences from the string.
func String(s string) string {
	return re.ReplaceAllString(s, "")
}

// Writer wraps an [io.Writer] and removes ANSI escape sequences from its output.
type StripAnsiWriter struct {
	w io.Writer
}

// NewStripAnsiWriter creates a new [Writer].
func NewStripAnsiWriter(w io.Writer) *StripAnsiWriter {
	return &StripAnsiWriter{w: w}
}

// Write removes ANSI escape sequences and writes to the underlying writer.
func (w *StripAnsiWriter) Write(p []byte) (n int, err error) {
	return w.w.Write(Bytes(p))
}

// StripAnsiReader wraps an [io.Reader] and removes ANSI escape sequences from its output.
type StripAnsiReader struct {
	r io.Reader
}

// NewStripAnsiReader creates a new [Reader].
func NewStripAnsiReader(r io.Reader) *StripAnsiReader {
	return &StripAnsiReader{r: r}
}

// Read reads from the underlying reader and removes ANSI escape sequences.
func (r *StripAnsiReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err != nil {
		return n, err
	}

	cleaned := Bytes(p[:n])
	copy(p, cleaned)

	return len(cleaned), nil
}
