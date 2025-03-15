// Copyright (c) 2020, The Garble Authors.
// See LICENSE for licensing information.

package literals

import (
	"go/ast"
	"go/token"
	mathrand "math/rand"
	"slices"

	ah "mvdan.cc/garble/internal/asthelper"
)

type simple struct{}

// check that the obfuscator interface is implemented
var _ obfuscator = simple{}

func (simple) obfuscate(obfRand *mathrand.Rand, data []byte) (*ast.BlockStmt, []externalKey) {
	key := make([]byte, len(data))
	obfRand.Read(key)

	originalKey := make([]byte, len(key))
	copy(originalKey, key)

	externalKeys := randExternalKeys(obfRand, len(data)+len(key))

	var (
		keyStmts, dataStmts []ast.Stmt
	)

	for _, k := range externalKeys {
		idx := obfRand.Intn(len(data))
		op := randOperator(obfRand)

		if obfRand.Intn(2) == 0 {
			data[idx] = evalOperator(op, data[idx], k.Byte())
			dataStmts = append(dataStmts, &ast.AssignStmt{
				Lhs: []ast.Expr{ah.IndexExpr("data", ah.IntLit(idx))},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{operatorToReversedBinaryExpr(op, ah.IndexExpr("data", ah.IntLit(idx)), k.Var())},
			})
		} else {
			originalKey[idx] = evalOperator(op, originalKey[idx], k.Byte())
			keyStmts = append(keyStmts, &ast.AssignStmt{
				Lhs: []ast.Expr{ah.IndexExpr("key", ah.IntLit(idx))},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{operatorToReversedBinaryExpr(op, ah.IndexExpr("key", ah.IntLit(idx)), k.Var())},
			})
		}
	}

	slices.Reverse(keyStmts)
	slices.Reverse(dataStmts)

	op := randOperator(obfRand)
	for i, b := range key {
		data[i] = evalOperator(op, data[i], b)
	}

	return ah.BlockStmt(
		&ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("key")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ah.DataToByteSlice(originalKey)},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("data")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ah.DataToByteSlice(data)},
		},
		ah.BlockStmt(keyStmts...),
		&ast.RangeStmt{
			Key:   ast.NewIdent("i"),
			Value: ast.NewIdent("b"),
			Tok:   token.DEFINE,
			X:     ast.NewIdent("key"),
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{ah.IndexExpr("data", ast.NewIdent("i"))},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{operatorToReversedBinaryExpr(op, ah.IndexExpr("data", ast.NewIdent("i")), ast.NewIdent("b"))},
				},
			}},
		},
		ah.BlockStmt(dataStmts...),
	), externalKeys
}
