package parse

import (
	"github.com/stretchr/testify/assert"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"testing"
)

func TestStatementUsesIdentifier(t *testing.T) {
	tests := map[string]struct {
		statement     string
		identifier    string
		hasIdentifier bool
	}{
		"simple assignment rhs with": {"x := y", "y", true},
		"simple assignment lhs with": {"y := x", "y", true},
		"simple assignment without":  {"z := x", "y", false},

		"multiple assignment rhs with":    {"x, z := q, y", "y", true},
		"multiple assignment lhs with":    {"y, q := x, p", "y", true},
		"multiple assignment lhs without": {"z := x", "y", false},

		"empty statement": {";", "y", false},

		"closure": {"func (a string) { fmt.Println(y) }", "y", true},
		"shadows": {"func (x, y string) { fmt.Println(y) }", "y", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmp, err := ioutil.TempFile("", "")
			assert.NoError(t, err)

			t.Logf("temp file name: %v", tmp.Name())

			_, err = tmp.WriteString("package main\n\nfunc main() {\n")
			assert.NoError(t, err)

			_, err = tmp.WriteString(tc.statement)
			assert.NoError(t, err)

			_, err = tmp.WriteString("\n}\n")
			assert.NoError(t, err)

			err = tmp.Sync()
			assert.NoError(t, err)

			err = tmp.Close()
			assert.NoError(t, err)

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tmp.Name(), nil, 0)
			assert.NoError(t, err)

			funcDecl := f.Decls[0].(*ast.FuncDecl)

			assert.Equal(t, tc.hasIdentifier, statementUsesIdent(funcDecl.Body, tc.identifier))
		})

	}

}
