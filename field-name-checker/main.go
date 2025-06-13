package main

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var Analyzer = &analysis.Analyzer{
	Name: "tfschemafield",
	Doc:  "checks for invalid schema field references in HasChange/Get/Set/etc.",
	Run:  run,
}

func getCompositeLiteralKey(n ast.Node) (string, bool) {
	/*
		map[string]*pluginsdk.Schema{
				"name": {   <--- Look for composite literal keys
					Type:         pluginsdk.TypeString,
					Required:     true,
				},
	*/
	if kv, ok := n.(*ast.KeyValueExpr); ok {
		if bl, ok := kv.Key.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			return strings.Trim(bl.Value, `"`), true
		}
	}
	return "", false
}

func getStringLiteralInAssignment(n ast.Node) (string, bool) {
	/*
		if !features.FivePointOh() {
			s["slice_service_type"] = &pluginsdk.Schema{ <--- Look for string literals in assignment statements
				Type:          pluginsdk.TypeInt,
				Optional:      true,
			}
	*/
	if assign, ok := n.(*ast.AssignStmt); ok {
		for _, lhs := range assign.Lhs {
			if idx, ok := lhs.(*ast.IndexExpr); ok {
				if key, ok := idx.Index.(*ast.BasicLit); ok && key.Kind == token.STRING {
					return strings.Trim(key.Value, `"`), true
				}
			}
		}
	}
	return "", false
}

func getFunctionName(n ast.Node) string {
	/*
		Look for function calls like:
			if d.HasChange("name") {
		Returns "HasChange" in this case
	*/
	if call, ok := n.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		} else if ident, ok := call.Fun.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func getFunctionArguments(n ast.Node) []ast.Expr {
	/*
		For function calls, we need to extract the arguments:
			d.HasChange("name")
	*/
	if call, ok := n.(*ast.CallExpr); ok {
		return call.Args
	}
	return nil
}

func getArgumentComponents(arg ast.Expr) []string {
	/*
		For arguments like "block.0.inner", we need to split by dot:
			d.HasChange("block.0.inner")
	*/
	if bl, ok := arg.(*ast.BasicLit); ok && bl.Kind == token.STRING {
		return strings.Split(strings.Trim(bl.Value, `"`), ".")
	}
	return nil
}

func run(pass *analysis.Pass) (interface{}, error) {
	schemaKeysSet := make(map[string]struct{})
	argumentComponentSet := make(map[string][]token.Pos)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if key, ok := getCompositeLiteralKey(n); ok {
				schemaKeysSet[key] = struct{}{}
			}
			if str, ok := getStringLiteralInAssignment(n); ok {
				schemaKeysSet[str] = struct{}{}
			}
			functionName := getFunctionName(n)
			if slices.Contains([]string{"HasChange", "HasChanges", "Get", "Set", "GetOk", "GetChange"}, functionName) {
				for _, arg := range getFunctionArguments(n) {
					for _, comp := range getArgumentComponents(arg) {
						// If the component can parse to an integer, skip it
						if _, err := strconv.Atoi(comp); err == nil || comp == "#" {
							continue
						}
						argumentComponentSet[comp] = append(argumentComponentSet[comp], arg.Pos())
					}
					if functionName != "HasChanges" {
						break
					}
				}
			}
			return true
		})
	}

	for k, v := range argumentComponentSet {
		if _, found := schemaKeysSet[k]; !found {
			for _, pos := range v {
				pass.Reportf(pos, "schema field/component '%s' not found in package", k)
			}
		}
	}

	return nil, nil
}

func main() {
	singlechecker.Main(Analyzer)
}
