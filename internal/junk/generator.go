package junk

import (
	"fmt"
	"go/ast"
	"go/token"
	mathrand "math/rand"
	ah "mvdan.cc/garble/internal/asthelper"
)

type Block struct {
	Vars  []ast.Expr
	Block *ast.BlockStmt
}

type generator interface {
	Generate(vars []ast.Expr) *Block
}

type FlowBlock struct {
	Block
	Body *ast.BlockStmt
}

type flowGenerator interface {
	Generate(vars []ast.Expr) *FlowBlock
}

var flowGenerators = []flowGenerator{&FlowForGenerator{}}
var generators = []generator{&FuncGenerator{}, &ModifyGenerator{}}

func generateJunkCode(vars []ast.Expr, flowCount, count int) *ast.BlockStmt {
	result := ah.BlockStmt()

	for i := 0; i < flowCount; i++ {
		flowGenerator := flowGenerators[mathrand.Intn(len(flowGenerators))]
		flowBlock := flowGenerator.Generate(vars)

		allVars := make([]ast.Expr, len(vars))
		copy(allVars, vars)

		if flowBlock.Vars != nil {
			allVars = append(allVars, flowBlock.Vars...)
		}

		for i := 0; i < count; i++ {
			generator := generators[mathrand.Intn(len(generators))]
			generateResult := generator.Generate(allVars)
			if generateResult.Vars != nil {
				allVars = append(allVars, generateResult.Vars...)
			}
			flowBlock.Body.List = append(flowBlock.Body.List, generateResult.Block.List...)
		}

		result.List = append(result.List, flowBlock.Block.Block.List...)
	}

	return result
}

func generateUniqueName() string {
	return fmt.Sprintf("__x%d", mathrand.Uint32())
}

func randomToken(tokens ...token.Token) token.Token {
	return tokens[mathrand.Intn(len(tokens))]
}

func randomExpr(tokens []ast.Expr) ast.Expr {
	return tokens[mathrand.Intn(len(tokens))]
}

func randomExprOrConst(exprs []ast.Expr) ast.Expr {
	return randomExpr(append([]ast.Expr{
		ah.CallExpr(ast.NewIdent("int32"), ah.IntLit(int(mathrand.Int31()))),
	}, exprs...))
}
