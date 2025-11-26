package lang

import (
	"context"
	"testing"
)

func TestJavascript(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		script   string
		expected string
	}{
		{`1 + 2`, "3"},
		{`
function sum(a, b) {
    return a + b;
}
sum(2, 2)
		`, "4"},
	}

	for _, tt := range tests {
		result, err := Javascript(ctx, tt.script)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result != tt.expected {
			t.Errorf("Script: %s - Expected: %s, got: %v", tt.script, tt.expected, result)
		}
		t.Logf("got: %s", result)
	}
}

func TestJavascriptFetch(t *testing.T) {
	script := `
function fibonacci(n) {
  if (n <= 0) return 0;
  if (n === 1) return 1;

  var a = 0;
  var b = 1;
  var fib = 1;
  
  for (var i = 2; i <= n; i++) {
    fib = a + b;
    a = b;
    b = fib;
  }
  
  return fib;
}

fibonacci(10);
`
	ctx := context.Background()
	result, err := Javascript(ctx, script)
	if err != nil {
		t.FailNow()
	}
	t.Logf("got: %v", result)
}
