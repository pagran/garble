package junk

import (
	"go/ast"
	mathrand "math/rand"
	ah "mvdan.cc/garble/internal/asthelper"
)

type FuncGenerator struct {
}

var _ generator = &FuncGenerator{}

func (*FuncGenerator) Generate(vars []ast.Expr) *Block {
	supportedFunctions := []string{"print", "println"}
	targetFunc := supportedFunctions[mathrand.Intn(len(supportedFunctions))]

	argsCount := len(vars) / 2
	if argsCount < 1 {
		argsCount = 1
	}

	args := make([]ast.Expr, argsCount)
	for i := 0; i < argsCount; i++ {
		args[i] = randomExprOrConst(vars)
	}
	callExpr := ah.CallExpr(ast.NewIdent(targetFunc), args...)
	return &Block{Block: ah.BlockStmt(ah.ExprStmt(callExpr))}
}
