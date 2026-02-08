package md

import (
	"fmt"
	"os"
	"testing"
)

func TestDepsAndArgs(t *testing.T) {
	c, err := os.ReadFile("./testdata/fixtures/deps_and_args.md")
	if err != nil {
		t.Fatal(err)
	}
	tf, err := Parse(string(c))
	if err != nil {
		t.Fatal(err)
	}

	tasks := tf.Tasks["default"]
	if len(tasks) == 0 {
		t.Fatal("no tasks")
	}

	task := tasks[0]
	fmt.Printf("Task: %s\n", task.Name)
	fmt.Printf("Display: %s\n", task.Display)
	fmt.Printf("Description: %s\n", task.Description)
	fmt.Printf("MimeType: %s\n", task.MimeType)
	fmt.Printf("Content: %s\n", task.Content)
	fmt.Printf("Dependencies: %d\n", len(task.Dependencies))
	for i, dep := range task.Dependencies {
		fmt.Printf("  Dep %d: %s\n", i, dep.Name)
	}
	fmt.Printf("Arguments: %v\n", task.Arguments)
}
