package lang

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

func Exec(dir string, env []string, name string, args []string) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Dir = dir
	cmd.Env = env
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func Compile(source string, bin string) error {
	args := []string{"build", "-o", bin}
	_, stderr, err := Exec(source, nil, "go", args)
	if err != nil {
		return fmt.Errorf("build failed: %v, stderr: %s", err, stderr)
	}
	if chmodErr := os.Chmod(bin, 0755); chmodErr != nil {
		return fmt.Errorf("chmod failed: %v", chmodErr)
	}
	return nil
}

func GoRun(source, bin string) (string, string, error) {
	if err := Compile(source, bin); err != nil {
		return "", "", fmt.Errorf("compilation failed: %w", err)
	}
	stdout, stderr, err := Exec(filepath.Dir(bin), nil, "./"+filepath.Base(bin), nil)
	if err != nil {
		return stdout, stderr, fmt.Errorf("execution failed: %w", err)
	}
	return stdout, stderr, nil
}

func goVersion() string {
	version := runtime.Version()
	cleanVersion := strings.TrimPrefix(version, "go")
	versionParts := strings.Split(cleanVersion, ".")
	goVersion := versionParts[0] + "." + versionParts[1]
	return goVersion
}

func ApplyGoRun(base string, code string, name string, envs map[string]any) (string, string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	subDir := filepath.Join(base, fmt.Sprintf("run_%d", r.Intn(1000000)))
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create subdirectory: %w", err)
	}

	tmpl, err := template.New("code").Parse(code)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse template: %w", err)
	}

	var processedCode bytes.Buffer
	if err := tmpl.Execute(&processedCode, envs); err != nil {
		return "", "", fmt.Errorf("failed to apply template: %w", err)
	}

	codeFilePath := filepath.Join(subDir, fmt.Sprintf("%s.go", name))
	if err := os.WriteFile(codeFilePath, processedCode.Bytes(), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write code to file: %w", err)
	}

	goModContent := []byte("module temp\n\ngo " + goVersion() + "\n")
	goModPath := filepath.Join(subDir, "go.mod")
	if err := os.WriteFile(goModPath, goModContent, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write go.mod: %w", err)
	}

	_, stderr, err := Exec(subDir, nil, "go", []string{"mod", "tidy"})
	if err != nil {
		return "", stderr, fmt.Errorf("go mod tidy failed: %w, stderr: %s", err, stderr)
	}

	stdout, stderr, err := GoRun(subDir, filepath.Join(subDir, name))
	if err != nil {
		return stdout, stderr, fmt.Errorf("execution failed: %w", err)
	}

	return stdout, stderr, nil
}

type GoRunner struct {
	Base string
}

func NewGoRunner(base string) *GoRunner {
	return &GoRunner{
		Base: base,
	}
}

func (gr *GoRunner) Run(code string, name string, args map[string]any) (string, string, error) {
	return ApplyGoRun(gr.Base, code, name, args)
}
