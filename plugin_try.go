package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"strings"
)

const (
	NONE = iota
	FATAL
	RETURN
	FUNC
)

func StripTry(node *ast.CallExpr) *ast.CallExpr {
	if len(node.Args) == 0 {
		log.Fatal("empty try() statement")
	}
	return node.Args[0].(*ast.CallExpr)
}

func GetTryForce(node *ast.CallExpr) string {
	args := node.Args
	if len(args) > 1 {
		if v, ok := args[1].(*ast.Ident); ok {
			switch v.String() {
			case "FATAL":
				fallthrough
			case "RETURN":
				return v.String()
			}
		}
	}
	return ""
}

func Selector(name string) ast.Expr {
	sections := strings.Split(name, ".")
	get := func(n int) *ast.Ident {
		return ast.NewIdent(sections[n])
	}
	if len(sections) == 1 {
		return get(0)
	}
	cur := &ast.SelectorExpr{get(0), get(1)}
	for i := 2; i < len(sections); i++ {
		cur = &ast.SelectorExpr{cur, get(i)}
	}
	return cur
}

func FuncReturnsError(decl *ast.FuncDecl) bool {
	results := decl.Type.Results
	result := false
	if results != nil {
		for _, v := range results.List {
			result = v.Type.(*ast.Ident).String() == "error"
		}
	}
	return result
}

func ParentFunc(tree *NodeTree) *ast.FuncDecl {
	funcs := tree.SearchParent(func(n *NodeTree) bool {
		_, ok := n.Node.(*ast.FuncDecl)
		return ok
	})
	// TODO: what will this do if the sender closes the channel without sending?
	decl := (<-funcs).Node.(*ast.FuncDecl)
	for _ = range funcs {
	}
	return decl
}

func BlockDefinesSymbol(block []ast.Stmt, name string) bool {
	for _, stmt := range block {
		fmt.Printf("%T\n", stmt)
	}
	return false
}

func BuildFuncReturn(f *ast.FuncDecl) []ast.Stmt {
	results := f.Type.Results
	var decls []ast.Stmt
	var values []ast.Expr
	if results != nil {
		// TODO: handle errors out of order
		// honestly, just have a "make empty return values" function, which also handles error
		for i, v := range results.List {
			name := v.Type.(*ast.Ident).String()
			var ident string
			switch name {
			case "error":
				ident = "err"
			default:
				ident = fmt.Sprintf("ret%d", i)
			}
			id := ast.NewIdent(ident)
			if ident != "err" {
				decls = append(decls, &ast.DeclStmt{
					&ast.GenDecl{nil, 0, token.VAR, 0,
						[]ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{id}, Type: v.Type}}, 0}})
			}
			values = append(values, ast.NewIdent(ident))
		}
	}
	return append(decls, &ast.ReturnStmt{0, values})
}

func BuildTryBlock(f *ast.FuncDecl, node *ast.CallExpr) (block []ast.Stmt) {
	var values []ast.Expr
	var funcLit *ast.FuncLit
	method := NONE
	if args := node.Args; len(args) > 1 {
		// TODO: assert args[0] is a CallExpr
		for i, arg := range args[1:] {
			switch v := arg.(type) {
			case *ast.Ident:
				id := v.String()
				if (id == "FATAL" || id == "RETURN") && i > 0 {
					log.Printf("Warning: ignored late '%s' in try()\n", id)
					continue
				}
				switch id {
				case "FATAL":
					method = FATAL
				case "RETURN":
					method = RETURN
				default:
					values = append(values, arg)
				}
			case *ast.BasicLit:
				values = append(values, arg)
			case *ast.FuncLit:
				if method == NONE {
					method = FUNC
				} else {
					log.Print("Warning: try() cannot use function literal with explicit FATAL or RETURN")
				}
				funcLit = v
				remaining := len(args) - i - 2
				if remaining > 0 {
					log.Printf("Warning: try() ignoring (%d) arguments after function literal\n", remaining)
					break
				}
			default:
				ast.Print(nil, arg)
			}
		}
	}
	if method == NONE {
		method = FATAL
		if FuncReturnsError(f) {
			method = RETURN
		}
	}
	var logFunc ast.Expr
	if method == FATAL {
		logFunc = Selector("log.Fatal")
	} else {
		logFunc = Selector("log.Print")
	}
	switch method {
	case FUNC:
		if len(values) == 0 {
			break
		}
		fallthrough
	case FATAL:
		fallthrough
	case RETURN:
		values = append(values, ast.NewIdent("err"))
		// <logFunc>(values..., err)
		logStmt := &ast.ExprStmt{&ast.CallExpr{logFunc, 0, values, 0, 0}}
		block = append(block, logStmt)
	}
	if method == FUNC {
		block = append(block, funcLit.Body.List...)
	} else if method == RETURN {
		block = append(block, BuildFuncReturn(f)...)
	}
	return
}

func AppendTryBlock(block []ast.Stmt, node ast.Node, errBlock []ast.Stmt) []ast.Stmt {
	var try, call *ast.CallExpr
	var assign *ast.AssignStmt
	_err, _nil := ast.NewIdent("err"), ast.NewIdent("nil")
	// err != nil
	errNil := &ast.BinaryExpr{_err, 0, token.NEQ, _nil}
	// if errNil { stmt }
	ifBlock := &ast.IfStmt{0, nil, errNil, &ast.BlockStmt{0, errBlock, 0}, nil}

	switch n := node.(type) {
	case *ast.AssignStmt:
		assign = n
		try = n.Rhs[0].(*ast.CallExpr)
	case *ast.ExprStmt:
		try = n.X.(*ast.CallExpr)
	default:
		log.Fatalf("unhandled try() node type: %T\n", node)
	}
	if assign == nil {
		assign = &ast.AssignStmt{nil, 0, token.ASSIGN, nil}
	}
	call = StripTry(try)
	assign.Rhs = []ast.Expr{call}
	assign.Lhs = append(assign.Lhs, _err)

	block = append(block, assign)
	block = append(block, ifBlock)
	return block
}

func GetTryCall(node ast.Node) *ast.CallExpr {
	switch n := node.(type) {
	// ^a := try(func)$
	case *ast.AssignStmt:
		if len(n.Rhs) == 1 {
			return GetTryCall(n.Rhs[0])
		}
	case *ast.ExprStmt:
		return GetTryCall(n.X)
	// ^try(func)$
	case *ast.CallExpr:
		switch t := n.Fun.(type) {
		case *ast.Ident:
			if t.String() == "try" {
				return n
			}
		}
	default:
		// log.Printf("unhandled type: %T\n", node)
	}
	return nil
}

func EnsureNoTry(f *ast.File) {
	idents := FilterAst(f, func(n ast.Node) bool {
		_, ok := n.(*ast.Ident)
		return ok
	})
	used := false
	for tree := range idents {
		ident := tree.Node.(*ast.Ident)
		if ident.String() == "try" {
			if tree.Parent != nil {
				if _, ok := tree.Parent.Node.(*ast.CallExpr); ok {
					continue
				}
			}
			used = true
		}
	}
	if used {
		fmt.Println("Warning: `try` used in code.")
	}
}

func ParseTry(fset *token.FileSet, f *ast.File) {
	blocks := FilterAst(f, func(n ast.Node) bool {
		return GetBlock(n) != nil
	})
	for tree := range blocks {
		parent := ParentFunc(tree)

		b := GetBlock(tree.Node)
		var block []ast.Stmt
		madeTry := false
		for _, v := range *b {
			if try := GetTryCall(v); try != nil {
				madeTry = true
				block = AppendTryBlock(block, v, BuildTryBlock(parent, try))
			} else {
				block = append(block, v)
			}
		}
		if madeTry {
			varErr := &ast.ValueSpec{nil, []*ast.Ident{ast.NewIdent("err")}, ast.NewIdent("error"), nil, nil}
			defineErr := &ast.GenDecl{nil, 0, token.VAR, 0, []ast.Spec{varErr, nil}, 0}
			stmt := []ast.Stmt{&ast.DeclStmt{defineErr}}
			block = append(stmt, block...)
		}
		*b = block
	}
	EnsureNoTry(f)
}
