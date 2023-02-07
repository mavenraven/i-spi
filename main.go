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
	err = lintPackages(packages)
	fmt.Println(err)
	fmt.Printf("done")
}

func lintPackages(pkgs map[string]*ast.Package) error {
	typesInPkg := make(map[string]*ast.TypeSpec)
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
						for _, spec := range genDecl.Specs {
							typeSpec := spec.(*ast.TypeSpec)
							typesInPkg[typeSpec.Name.Name] = typeSpec
						}
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
				case *ast.FuncDecl:
					funcDecl := decl.(*ast.FuncDecl)
					if !ast.IsExported(funcDecl.Name.Name) {
						continue
					}

					if funcDecl.Recv == nil {
						return fmt.Errorf("function %v is exported and not a method on pkg", funcDecl.Name.Name)
					}

					if len(funcDecl.Recv.List) != 1 {
						panic("bug with length of receiver list")
					}

					receiver := funcDecl.Recv.List[0]

					if len(receiver.Names) != 1 {
						panic("bug with length of receiver names list")
					}

					if _, ok := receiver.Type.(*ast.StarExpr); ok {
						return fmt.Errorf("function %v declared with a pointer receiver", funcDecl.Name.Name)
					}

					recvTypeIdentifier, ok := receiver.Type.(*ast.Ident)
					if !ok {
						panic("recvIdentifier could not be cast")
					}

					panic(recvTypeIdentifier)

					if funcDecl.Recv.List[0].Names[0].Name != "p" {
						return fmt.Errorf("function %v should have a receiver named p", funcDecl.Name.Name)
					}

					if funcDecl.Type.TypeParams != nil {
						return fmt.Errorf("function %v must not be generic", funcDecl.Name.Name)
					}

					for _, param := range funcDecl.Type.Params.List {
						t := param.Type
						panic(t)

					}

				default:
					panic("unhandled declaration type")
				}
			}
		}
	}
	return nil
}
