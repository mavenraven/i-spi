package parse

import (
	"github.com/stretchr/testify/assert"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"testing"
)

func TestStatementAccessesValueInIdentifier(t *testing.T) {
	tests := map[string]struct {
		statement      string
		identifier     string
		usesIdentifier bool
	}{
		"simple assignment with":    {"x := y", "y", true},
		"simple assignment without": {"z := x", "y", false},

		"multiple assignment with":    {"x, z := q, y", "y", true},
		"multiple assignment without": {"z := x", "y", false},

		"empty statement": {";", "y", false},

		"closure":      {"func (a string) { fmt.Println(y) }", "y", true},
		"func without": {"func (a string) { fmt.Println(z) }", "y", false},
		"shadows":      {"func (x, y string) { fmt.Println(y) }", "y", false},

		"const with":    {"const x = y", "y", true},
		"const without": {"const x = z", "y", false},

		"var with":    {"var x = y", "y", true},
		"var without": {"var x = z", "y", false},

		"type": {"type x struct {}", "y", false},

		"labeled with":    {"Hello:\nfmt.Println(y)", "y", true},
		"labeled without": {"Hello:\nfmt.Println(z)", "y", false},

		"send with rhs": {"c <- y", "y", true},
		"send with lhs": {"y <- x", "y", true},
		"send without":  {"c <- z", "y", false},

		"inc with":    {"y++", "y", true},
		"inc without": {"z++", "y", false},

		"go with":    {"go func() { fmt.Println(y) }()", "y", true},
		"go without": {"go func() { fmt.Println(z) }()", "y", false},

		"call with identifier as receiver":    {"y.Hello()", "y", true},
		"call without identifier as receiver": {"z.y.Hello()", "y", false},
		"call with identifier as an argument": {"z.x.Hello(y).Bye()", "y", true},

		"defer with":    {"defer fmt.Println(y)", "y", true},
		"defer without": {"defer fmt.Println(z)", "y", false},
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

			assert.Equal(t, tc.usesIdentifier, statementAccessesValueInIdentifier(funcDecl.Body, tc.identifier))
		})

	}

}
