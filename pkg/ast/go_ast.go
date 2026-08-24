package ast

import (
	goast "go/ast"
	"go/parser"
	"go/token"
)

// GoParser implements AST scope extraction for Go source files.
type GoParser struct{}

// GetRelevantScopes extracts function declarations, method receivers, and type specs enclosing changed lines.
func (p *GoParser) GetRelevantScopes(code string, changedLines []int) []ScopeRange {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return nil
	}

	var scopes []ScopeRange
	changedSet := make(map[int]bool)
	for _, l := range changedLines {
		changedSet[l] = true
	}

	goast.Inspect(file, func(n goast.Node) bool {
		if n == nil {
			return true
		}

		switch decl := n.(type) {
		case *goast.FuncDecl:
			start := fset.Position(decl.Pos()).Line
			end := fset.Position(decl.End()).Line
			if p.hasChangedLine(start, end, changedSet) {
				scopes = append(scopes, ScopeRange{StartLine: start, EndLine: end})
			}
			return false // Don't inspect inside function again to keep outermost scope

		case *goast.GenDecl:
			if decl.Tok == token.TYPE || decl.Tok == token.CONST || decl.Tok == token.VAR {
				start := fset.Position(decl.Pos()).Line
				end := fset.Position(decl.End()).Line
				if p.hasChangedLine(start, end, changedSet) {
					scopes = append(scopes, ScopeRange{StartLine: start, EndLine: end})
				}
			}
			return false
		}

		return true
	})

	return scopes
}

func (p *GoParser) hasChangedLine(start, end int, changedSet map[int]bool) bool {
	for l := start; l <= end; l++ {
		if changedSet[l] {
			return true
		}
	}
	return false
}
