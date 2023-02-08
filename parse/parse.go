package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
)

func parse() {

	f, _ := ioutil.TempFile("", "")
	fmt.Println(f.Name())

	f.WriteString(`
func main() {
    fmt.Println("hello world")
}
`)
	f.Sync()
	f.Close()

	fset2 := token.NewFileSet()
	expr, err := parser.ParseFile(fset2, f.Name(), nil, 0)
	fmt.Println(expr)

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

func statementAccessesValueInIdentifier(stmt ast.Stmt, ident string) bool {
	switch stmt.(type) {
	case *ast.BadStmt:
		panic("unreachable")
	case *ast.DeclStmt:
		genDecl := stmt.(*ast.DeclStmt).Decl.(*ast.GenDecl)
		for _, spec := range genDecl.Specs {
			switch spec.(type) {
			case *ast.ValueSpec:
				for _, val := range spec.(*ast.ValueSpec).Values {
					if expressionAccessValueInIdentifier(val, ident) {
						return true
					}

					return false
				}
			case *ast.TypeSpec:
				return false
			case *ast.ImportSpec:
				panic("unreachable")
			}
		}
	case *ast.EmptyStmt:
		return false
	case *ast.LabeledStmt:
		labeledStmt := stmt.(*ast.LabeledStmt)
		panic(labeledStmt)
	case *ast.ExprStmt:
		exprStmt := stmt.(*ast.ExprStmt)
		return expressionAccessValueInIdentifier(exprStmt.X, ident)
	case *ast.SendStmt:
		sendStmt := stmt.(*ast.SendStmt)
		panic(sendStmt)
	case *ast.IncDecStmt:
		incDecStmt := stmt.(*ast.IncDecStmt)
		panic(incDecStmt)
	case *ast.AssignStmt:
		assignStmnt := stmt.(*ast.AssignStmt)
		for _, expr := range assignStmnt.Lhs {
			if expressionAccessValueInIdentifier(expr, ident) {
				return true
			}
		}

		for _, expr := range assignStmnt.Rhs {
			if expressionAccessValueInIdentifier(expr, ident) {
				return true
			}
		}

		return false
	case *ast.GoStmt:
		goStmt := stmt.(*ast.GoStmt)
		panic(goStmt)
	case *ast.DeferStmt:
		deferStmt := stmt.(*ast.DeferStmt)
		panic(deferStmt)
	case *ast.ReturnStmt:
		returnStmt := stmt.(*ast.ReturnStmt)
		panic(returnStmt)
	case *ast.BranchStmt:
		branchStmt := stmt.(*ast.BranchStmt)
		panic(branchStmt)
	case *ast.BlockStmt:
		blockStmt := stmt.(*ast.BlockStmt)
		for _, stmt := range blockStmt.List {
			if statementAccessesValueInIdentifier(stmt, ident) {
				return true
			}
		}
		return false
	case *ast.IfStmt:
		ifStmt := stmt.(*ast.IfStmt)
		panic(ifStmt)
	case *ast.CaseClause:
		caseClause := stmt.(*ast.CaseClause)
		panic(caseClause)
	case *ast.SwitchStmt:
		switchStmt := stmt.(*ast.SwitchStmt)
		panic(switchStmt)
	case *ast.TypeSwitchStmt:
		typeSwitchStmt := stmt.(*ast.TypeSwitchStmt)
		panic(typeSwitchStmt)
	case *ast.CommClause:
		commClause := stmt.(*ast.CommClause)
		panic(commClause)
	case *ast.SelectStmt:
		selectStmt := stmt.(*ast.SelectStmt)
		panic(selectStmt)
	case *ast.ForStmt:
		forStmt := stmt.(*ast.ForStmt)
		panic(forStmt)
	case *ast.RangeStmt:
		rangeStmt := stmt.(*ast.RangeStmt)
		panic(rangeStmt)
	}
	return false
}

func expressionAccessValueInIdentifier(expr ast.Expr, identifier string) bool {
	switch expr.(type) {
	case *ast.BadExpr:
		badExpr := expr.(*ast.BadExpr)
		panic(badExpr)
	case *ast.Ident:
		ident := expr.(*ast.Ident)
		return ident.Name == identifier
	case *ast.Ellipsis:
		ellipsis := expr.(*ast.Ellipsis)
		panic(ellipsis)
	case *ast.BasicLit:
		basicLit := expr.(*ast.BasicLit)
		panic(basicLit)
	case *ast.FuncLit:
		funcLit := expr.(*ast.FuncLit)
		for _, param := range funcLit.Type.Params.List {
			for _, name := range param.Names {
				if name.Name == identifier {
					return false
				}
			}
		}
		return statementAccessesValueInIdentifier(funcLit.Body, identifier)
	case *ast.CompositeLit:
		compositeLit := expr.(*ast.CompositeLit)
		panic(compositeLit)
	case *ast.ParenExpr:
		parenExpr := expr.(*ast.ParenExpr)
		panic(parenExpr)
	case *ast.SelectorExpr:
		selectorExpr := expr.(*ast.SelectorExpr)
		panic(selectorExpr)
	case *ast.IndexExpr:
		indexExpr := expr.(*ast.IndexExpr)
		panic(indexExpr)
	case *ast.IndexListExpr:
		indexListExpr := expr.(*ast.IndexListExpr)
		panic(indexListExpr)
	case *ast.SliceExpr:
		sliceExpr := expr.(*ast.SliceExpr)
		panic(sliceExpr)
	case *ast.TypeAssertExpr:
		typeAssertExpr := expr.(*ast.TypeAssertExpr)
		panic(typeAssertExpr)
	case *ast.CallExpr:
		callExpr := expr.(*ast.CallExpr)
		for _, arg := range callExpr.Args {
			if expressionAccessValueInIdentifier(arg, identifier) {
				return true
			}
		}
		return false
	case *ast.StarExpr:
		starExpr := expr.(*ast.StarExpr)
		panic(starExpr)
	case *ast.UnaryExpr:
		unaryExpr := expr.(*ast.UnaryExpr)
		panic(unaryExpr)
	case *ast.BinaryExpr:
		binaryExpr := expr.(*ast.BinaryExpr)
		panic(binaryExpr)
	case *ast.KeyValueExpr:
		keyValueExpr := expr.(*ast.KeyValueExpr)
		panic(keyValueExpr)
	case *ast.ArrayType:
		arrayType := expr.(*ast.ArrayType)
		panic(arrayType)
	case *ast.StructType:
		structType := expr.(*ast.StructType)
		panic(structType)
	case *ast.FuncType:
		funcType := expr.(*ast.FuncType)
		panic(funcType)
	case *ast.InterfaceType:
		interfaceType := expr.(*ast.InterfaceType)
		panic(interfaceType)
	case *ast.MapType:
		mapType := expr.(*ast.MapType)
		panic(mapType)
	case *ast.ChanType:
		chanType := expr.(*ast.ChanType)
		panic(chanType)
	}
	return false
}
