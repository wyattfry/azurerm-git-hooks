package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func Test_getCompositeLiteralKey(t *testing.T) {
	testdata := []struct {
		input    string
		expected string
	}{
		{`map[string]*pluginsdk.Schema{"name": {}}`, "name"},
		{`map[string]*pluginsdk.Schema{"age": {}}`, "age"},
		{`map[string]*pluginsdk.Schema{"": {}}`, ""},
	}

	for _, tt := range testdata {
		t.Run(tt.input, func(t *testing.T) {
			found := false
			for _, n := range parseIntoNodes(tt.input) {
				if key, ok := getCompositeLiteralKey(n); ok && key == tt.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected '%s', but did not find it", tt.expected)
			}
		})
	}
}

func Test_getStringLiteralInAssignment(t *testing.T) {
	testdata := []struct {
		input    string
		expected string
	}{
		{`s["slice_service_type"] = &pluginsdk.Schema{}`, "slice_service_type"},
		{`s["another_key"] = &pluginsdk.Schema{}`, "another_key"},
	}

	for _, tt := range testdata {
		t.Run(tt.input, func(t *testing.T) {
			found := false
			src := fmt.Sprintf(`package main
func dummy() {
			%s
		}`, tt.input)
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "dummy" {
					for _, stmt := range fn.Body.List {
						if assign, ok := stmt.(*ast.AssignStmt); ok {
							if str, ok := getStringLiteralInAssignment(assign); ok && str == tt.expected {
								found = true
								break
							}
						}
					}
				}
			}
			if !found {
				t.Errorf("expected '%s', but did not find it", tt.expected)
			}
		})
	}
}

func Test_getFunctionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`if d.HasChange("name") {}`, "HasChange"},
		{`HasChange("name")`, "HasChange"},
		{`fmt.Println("hello")`, "Println"},
		{`someFunc()`, "someFunc"},
		{`notACall`, ""},
	}

	for _, tt := range tests {
		// Parse the statement/expression
		src := "package main\nfunc test() { " + tt.input + "\n}"
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "test.go", src, 0)
		if err != nil {
			t.Fatalf("ParseFile failed: %v", err)
		}

		// Find the first call expression in the function body
		found := false
		ast.Inspect(file, func(n ast.Node) bool {
			if fn := getFunctionName(n); fn != "" {
				found = true
				if fn != tt.expected {
					t.Errorf("for input %q, expected %q, got %q", tt.input, tt.expected, fn)
				}
				return false // stop after first call
			}
			return true
		})
		if !found && tt.expected != "" {
			t.Errorf("for input %q, expected %q, got nothing", tt.input, tt.expected)
		}
	}
}

func parseIntoNodes(exprSrc string) []ast.Node {
	expr, err := parser.ParseExpr(exprSrc)
	if err != nil {
		panic(err)
	}
	nodes := []ast.Node{}
	if cl, ok := expr.(*ast.CompositeLit); ok {
		for _, elt := range cl.Elts {
			nodes = append(nodes, elt)
		}
	}
	return nodes
}

func Test_getFunctionArguments(t *testing.T) {
	tests := []struct {
		input    string
		expected []string // arguments as source code strings
	}{
		{`foo("bar", 123)`, []string{`"bar"`, `123`}},
		{`d.HasChange("block.0.inner")`, []string{`"block.0.inner"`}},
		{`call()`, []string{}},
		{`notACall`, nil},
	}

	for _, tt := range tests {
		expr, err := parser.ParseExpr(tt.input)
		if err != nil {
			t.Fatalf("ParseExpr failed for %s: %v", tt.input, err)
		}
		args := getFunctionArguments(expr)
		if tt.expected == nil {
			if args != nil {
				t.Errorf("expected nil, got %v for input %q", args, tt.input)
			}
			continue
		}
		if len(args) != len(tt.expected) {
			t.Errorf("expected %d args, got %d for input %q", len(tt.expected), len(args), tt.input)
			continue
		}
		for i, arg := range args {
			got := exprToString(arg)
			if got != tt.expected[i] {
				t.Errorf("expected arg[%d]=%q, got %q for input %q", i, tt.expected[i], got, tt.input)
			}
		}
	}
}

// Helper to turn ast.Expr back into source code (for comparison)
func exprToString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		return v.Name
	default:
		return ""
	}
}

func Test_getArgumentComponents(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`"block.0.inner"`, []string{"block", "0", "inner"}},
		{`"foo.bar"`, []string{"foo", "bar"}},
		{`"foo"`, []string{"foo"}},
		{`123`, nil},         // Not a string literal
		{`bar`, nil},         // Not a string literal
		{`""`, []string{""}}, // Empty string
	}

	for _, tt := range tests {
		expr, err := parser.ParseExpr(tt.input)
		if err != nil {
			t.Fatalf("ParseExpr failed for %s: %v", tt.input, err)
		}
		got := getArgumentComponents(expr)
		if len(got) != len(tt.expected) {
			t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, got)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, got)
				break
			}
		}
	}
}
