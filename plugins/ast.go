package plugins

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
)

type NodeTree struct {
	Node     ast.Node
	Parent   *NodeTree
	Callback func(tree *NodeTree) bool
}

func (tree *NodeTree) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	sub := &NodeTree{node, tree, tree.Callback}
	if tree.Callback(sub) {
		return sub
	}
	return nil
}

func (tree *NodeTree) SearchParent(callback func(*NodeTree) bool) chan *NodeTree {
	ret := make(chan *NodeTree)
	go func() {
		cur := tree
		for cur.Parent != nil {
			cur = cur.Parent
			if callback(cur) {
				ret <- cur
			}
		}
		close(ret)
	}()
	return ret
}

func (tree *NodeTree) Print(fset *token.FileSet) {
	bytes, err := CodeBytes(fset, tree.Node)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", bytes)
}

func WalkAst(node ast.Node, callback func(tree *NodeTree) bool) {
	ast.Walk(&NodeTree{node, nil, callback}, node)
}

func CodeBytes(fset *token.FileSet, node interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func FilterAst(top ast.Node, callback func(n ast.Node) bool) chan *NodeTree {
	ret := make(chan *NodeTree)
	go func() {
		WalkAst(top, func(n *NodeTree) bool {
			if callback(n.Node) {
				ret <- n
			}
			return true
		})
		close(ret)
	}()
	return ret
}
