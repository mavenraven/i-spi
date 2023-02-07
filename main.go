package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	fset := token.NewFileSet()
	packages, err := parser.ParseDir(fset, "/home/user/spi/tictactoe/", nil, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func lintPackages(pkgs map[string]*ast.Package) error {

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				switch decl.(type) {
				case *ast.GenDecl:
					genDecl := decl.(*ast.GenDecl)

					switch genDecl.Tok {
					case token.IMPORT:
						continue
					case token.CONST:
						// TODO: make this a bit smarter
						return fmt.Errorf("package not allowed to export constants")
					case token.TYPE:
						continue
					case token.VAR:
						for _, spec := range genDecl.Specs {
							valueSpec := spec.(*ast.ValueSpec)

							for _, name := range valueSpec.Names {
								if !ast.IsExported(name.Name) {
									continue
								}

								if name.Name != "Pkg" {
									return fmt.Errorf("package %v exports variable %v", pkg.Name, name)
								}
								panic("hi")
							}
						}
					}
				default:
					fmt.Printf("skipped\n")
				}
			}
		}
	}
}
