package ast

import (
	"strings"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "go"},
		{"service.py", "python"},
		{"app.tsx", "typescript"},
		{"index.js", "javascript"},
		{"Controller.php", "php"},
		{"User.java", "java"},
		{"main.rs", "rust"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.filename)
		if got != tt.expected {
			t.Errorf("DetectLanguage(%q) = %q, expected %q", tt.filename, got, tt.expected)
		}
	}
}

func TestGoASTParser(t *testing.T) {
	code := `package main

import "fmt"

func CalculateTotal(items []int) int {
	sum := 0
	for _, item := range items {
		sum += item
	}
	return sum
}

func PrintGreeting(name string) {
	fmt.Println("Hello", name)
}
`

	parser := &GoParser{}
	// Change in CalculateTotal (line 7)
	scopes := parser.GetRelevantScopes(code, []int{7})

	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if scopes[0].StartLine != 5 || scopes[0].EndLine != 11 {
		t.Errorf("expected scope range [5, 11], got [%d, %d]", scopes[0].StartLine, scopes[0].EndLine)
	}
}

func TestGenericPythonASTParser(t *testing.T) {
	code := `import os

def authenticate_user(username, password):
    if not username:
        return False
    return True

def process_payment(amount):
    print("Processing payment of", amount)
`

	parser := &GenericParser{Language: "python"}
	scopes := parser.GetRelevantScopes(code, []int{5})

	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope for Python, got %d", len(scopes))
	}
	if scopes[0].StartLine != 3 {
		t.Errorf("expected start line 3, got %d", scopes[0].StartLine)
	}
}

func TestSliceFileContext(t *testing.T) {
	smallCode := `package main

func Hello() string {
	return "world"
}
`
	output := SliceFileContext(smallCode, "hello.go", []int{4})
	if !strings.Contains(output, "1: package main") {
		t.Errorf("expected decorated line numbers in output, got:\n%s", output)
	}
}
