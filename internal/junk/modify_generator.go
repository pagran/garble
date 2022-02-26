package junk

import (
	"go/ast"
	"go/token"
	ah "mvdan.cc/garble/internal/asthelper"
)

type ModifyGenerator struct {
}

var _ generator = &ModifyGenerator{}

func (*ModifyGenerator) Generate(vars []ast.Expr) *Block {
	assignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{randomExpr(vars)},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.BinaryExpr{
			X:  randomExpr(vars),
			Op: randomToken(token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.AND, token.OR, token.XOR),
			Y:  randomExprOrConst(vars),
		},
		},
	}
	return &Block{Block: ah.BlockStmt(assignStmt)}
}
