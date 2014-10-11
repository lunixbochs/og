package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
)

func StripTry(node *ast.CallExpr) *ast.CallExpr {
	return node.Args[0].(*ast.CallExpr)
}

func IsTryFatal(node *ast.CallExpr) bool {
	args := node.Args
	if len(args) > 1 {
		if v, ok := args[1].(*ast.Ident); ok && v.String() == "FATAL" {
			return true
		}
	}
	return false
}

func FuncReturnsError(decl *ast.FuncDecl) bool {
	results := decl.Type.Results
	result := false
	if results != nil {
		// TODO: handle errors out of order
		// honestly, just have a "make empty return values" function, which also handles error
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

func BuildTryBlock(f *ast.FuncDecl, node *ast.CallExpr) []ast.Stmt {
	var block []ast.Stmt
	if !FuncReturnsError(f) || IsTryFatal(node) {
		_err, _log, _fatal := ast.NewIdent("err"), ast.NewIdent("log"), ast.NewIdent("Fatal")
		// log.Fatal(err)
		logFatal := &ast.ExprStmt{&ast.CallExpr{&ast.SelectorExpr{_log, _fatal}, 0, []ast.Expr{_err}, 0, 0}}
		block = append(block, logFatal)
	} else {
		block = append(block, BuildFuncReturn(f)...)
	}
	return block
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
		assign = &ast.AssignStmt{nil, 0, token.DEFINE, nil}
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

func ExpandTry(fset *token.FileSet, f *ast.File) {
	blocks := FilterAst(f, func(n ast.Node) bool {
		_, ok := n.(*ast.BlockStmt)
		return ok
	})
	for tree := range blocks {
		parent := ParentFunc(tree)

		b := tree.Node.(*ast.BlockStmt)
		var block []ast.Stmt
		for _, v := range b.List {
			if try := GetTryCall(v); try != nil {
				block = AppendTryBlock(block, v, BuildTryBlock(parent, try))
			} else {
				block = append(block, v)
			}
		}
		b.List = block
	}
}
