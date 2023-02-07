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
	lintPackages(packages)
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

							for i, name := range valueSpec.Names {
								if !ast.IsExported(name.Name) {
									continue
								}

								if name.Name != "Pkg" {
									return fmt.Errorf("package %v exports variable %v", pkg.Name, name)
								}

								compositeLit, ok := valueSpec.Values[i].(*ast.CompositeLit)
								if !ok {
									return fmt.Errorf("Pkg variable does not equal pkg")
								}

								ident, ok := compositeLit.Type.(*ast.Ident)
								if !ok {
									return fmt.Errorf("Pkg variable does not equal pkg")
								}

								if ident.Name != "pkg" {
									return fmt.Errorf("Pkg variable does not equal pkg")
								}

								pkgType, ok := ident.Obj.Decl.(*ast.TypeSpec)
								if !ok {
									return fmt.Errorf("pkg is not a type")
								}

								if pkgType.TypeParams != nil {
									return fmt.Errorf("pkg must not have any fields")
								}

							}
						}
					}
				default:
					fmt.Printf("skipped\n")
				}
			}
		}
	}
	return nil
}
