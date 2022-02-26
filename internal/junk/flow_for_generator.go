package junk

import (
	"go/ast"
	"go/token"
	ah "mvdan.cc/garble/internal/asthelper"
)

type FlowForGenerator struct {
}

var _ flowGenerator = &FlowForGenerator{}

func (*FlowForGenerator) Generate(vars []ast.Expr) *FlowBlock {
	body := &ast.BlockStmt{}

	i := ast.NewIdent(generateUniqueName())

	forExpr := &ast.ForStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{i},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{randomExprOrConst(vars)},
		},
		Cond: &ast.BinaryExpr{
			X:  i,
			Op: randomToken(token.LSS, token.EQL, token.GTR, token.NEQ, token.LEQ, token.GEQ),
			Y:  randomExprOrConst(vars),
		},
		Post: &ast.IncDecStmt{
			X:   i,
			Tok: randomToken(token.INC, token.DEC),
		},
		Body: body,
	}

	return &FlowBlock{
		Block: Block{
			Vars:  []ast.Expr{i},
			Block: ah.BlockStmt(forExpr),
		},
		Body: body,
	}
}
