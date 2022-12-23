package link

import (
	"go/ast"
	"go/token"
	ah "mvdan.cc/garble/internal/asthelper"
)

func ApplyRuntimeMagicValue(basename string, magicValue int, file *ast.File) {
	if basename != "Xsymtab.go" {
		return
	}

	// Find `hdr.magic != 0xfffffff0` and update to random magicValue
	updateMagic := func(node ast.Node) bool {
		binExpr, ok := node.(*ast.BinaryExpr)
		if !ok || binExpr.Op != token.NEQ {
			return true
		}

		selectorExpr, ok := binExpr.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selectorExpr.Sel.Name != "magic" {
			return true
		}

		if _, ok := binExpr.Y.(*ast.BasicLit); !ok {
			return true
		}

		println("Updated value!!!")
		binExpr.Y = ah.IntLit(magicValue)
		return false
	}

	ast.Inspect(file, updateMagic)
}
