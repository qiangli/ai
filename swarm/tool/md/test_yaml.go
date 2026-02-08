package md

import (
	"fmt"
	"os"
)

func TestYAMLParsing() {
	source := `### Task Two

Second task with dependencies

---
dependencies:
  - task-one
---

` + "```bash\necho two\n```"

	tf, err := Parse(source)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		os.Exit(1)
	}

	tasks := tf.Tasks["default"]
	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		os.Exit(1)
	}

	task := tasks[0]
	fmt.Printf("Task: %s\n", task.Name)
	fmt.Printf("Description: %s\n", task.Description)
	fmt.Printf("Dependencies: %d\n", len(task.Dependencies))
	for i, dep := range task.Dependencies {
		fmt.Printf("  Dep %d: %s\n", i, dep.Name)
	}
	fmt.Printf("Content: %s\n", task.Content)
}
