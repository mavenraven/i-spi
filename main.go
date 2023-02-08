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

var primitiveTypes = map[string]struct{}{
	"bool": {},

	"uint8":  {},
	"uint16": {},
	"uint32": {},
	"uint64": {},

	"int8":  {},
	"int16": {},
	"int32": {},
	"int64": {},

	"float32": {},
	"float64": {},

	"complex64":  {},
	"complex128": {},

	"byte": {},
	"rune": {},

	"uint":    {},
	"int":     {},
	"uintptr": {},

	"string": {},
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

								pkgStruct, ok := pkgType.Type.(*ast.StructType)
								if !ok {
									return fmt.Errorf("pkg must be a struct a type")
								}

								if len(pkgStruct.Fields.List) != 0 {
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

					if recvTypeIdentifier.Name != "pkg" {
						return fmt.Errorf("function %v's receiver is not package", funcDecl.Name.Name)
					}

					if funcDecl.Recv.List[0].Names[0].Name != "p" {
						return fmt.Errorf("function %v should have a receiver named p", funcDecl.Name.Name)
					}

					if funcDecl.Type.TypeParams != nil {
						panic("this currently isn't possible, but we want to prevent exporting generic methods")
					}

					for _, param := range funcDecl.Type.Params.List {
						if len(param.Names) == 0 {
							panic("param names was empty somehow")
						}

						if err := isSerializable(param.Type); err != nil {
							return fmt.Errorf("unable to serialize param %v: %w", param.Names[0].Name, err)
						}
					}

				default:
					panic("unhandled declaration type")
				}
			}
		}
	}
	return nil
}

func isSerializable(expr ast.Expr) error {
	switch expr.(type) {
	case *ast.Ident:
		ident := expr.(*ast.Ident)
		if _, ok := primitiveTypes[ident.Name]; ok {
			return nil
		}

		if !ast.IsExported(ident.Name) {
			return fmt.Errorf("%v type is not exported", ident.Name)
		}

		objType, ok := ident.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			return fmt.Errorf("unable to cast to type spec")
		}

		return isSerializable(objType.Type)
	case *ast.ArrayType:
		arrayType := expr.(*ast.ArrayType)
		if arrayType.Len == nil {
			return fmt.Errorf("slices are not serialiazble as they contain a pointer")
		}

		return isSerializable(arrayType.Elt)
	case *ast.StarExpr:
		return fmt.Errorf("pointers are not serializable")
	case *ast.MapType:
		return fmt.Errorf("maps are not serialiazble as they contain a pointer")
	case *ast.FuncType:
		return fmt.Errorf("functions are not serialiazble")
	case *ast.InterfaceType:
		return fmt.Errorf("interfaces are not serialiazble")
	case *ast.ChanType:
		return fmt.Errorf("channels are not serialiazble")
	case *ast.StructType:
		structType := expr.(*ast.StructType)
		for _, f := range structType.Fields.List {
			if f.Tag != nil {
				return fmt.Errorf("structs with tags are currently not considered to be serializable")
			}

			for _, name := range f.Names {
				if !ast.IsExported(name.Name) {
					return fmt.Errorf("field %v is not exported", name.Name)
				}
			}

			err := isSerializable(f.Type)
			if err != nil {
				return err
			}

		}
	default:
		return fmt.Errorf("expression is not serializable")
	}
	return nil
}
