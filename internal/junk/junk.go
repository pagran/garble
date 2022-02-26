package junk

import (
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	mathrand "math/rand"
	ah "mvdan.cc/garble/internal/asthelper"
)

func generateJunk(bootstrapVar *ast.Ident, boostrapLen int) *ast.BlockStmt {
	var vars []ast.Expr
	for i := 0; i < boostrapLen; i++ {
		vars = append(vars, ah.IndexExpr(bootstrapVar.Name, ah.IntLit(i)))
	}
	return generateJunkCode(vars, 1, 1)
}

func isSupportedFunc(funcType *ast.FuncType) bool {
	return funcType == nil || funcType.Results.NumFields() == 0
}

// TODO: rewrite logic
func generateFakeKey(keys []int32) int32 {
	for {
		key := mathrand.Int31()
		unique := true
		for i := 0; i < len(keys); i++ {
			if keys[i] == key {
				unique = false
				break
			}
		}

		if unique {
			return key
		}
	}
}

func mutateBlockStmt(blockStmt *ast.BlockStmt, keys []int32, bootstrapVar *ast.Ident, funcType *ast.FuncType, count int) *ast.BlockStmt {
	if !isSupportedFunc(funcType) {
		return nil
	}

	keyIndex := mathrand.Intn(len(keys))
	key := keys[keyIndex]

	cases := ah.BlockStmt(&ast.CaseClause{
		List: []ast.Expr{ah.IntLit(int(key))},
		Body: []ast.Stmt{ah.BlockStmt(blockStmt)},
	})

	for i := 0; i < count; i++ {
		cases.List = append(cases.List, &ast.CaseClause{
			List: []ast.Expr{ah.IntLit(int(generateFakeKey(keys)))},
			Body: []ast.Stmt{generateJunk(bootstrapVar, len(keys))},
		})
	}
	mathrand.Shuffle(len(cases.List), func(i, j int) { cases.List[i], cases.List[j] = cases.List[j], cases.List[i] })

	newBlockStmt := ah.BlockStmt(
		&ast.SwitchStmt{
			Tag:  ah.IndexExpr(bootstrapVar.Name, ah.IntLit(keyIndex)),
			Body: cases,
		},
	)
	return newBlockStmt
}

func generateBootstrapVar() (keys []int32, bootstrapVar *ast.Ident, bootstrapVarDecl *ast.GenDecl) {
	keyCount := 1 + mathrand.Intn(10)
	keys = make([]int32, keyCount)

	var values []ast.Expr
	for i := 0; i < keyCount; i++ {
		keys[i] = mathrand.Int31()
		values = append(values, ah.IntLit(int(keys[i])))
	}
	bootstrapVarName := generateUniqueName()
	bootstrapVar = ast.NewIdent(bootstrapVarName)

	bootstrapVarDecl = &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{bootstrapVar},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.ArrayType{
							Len: ah.IntLit(keyCount),
							Elt: ast.NewIdent("int32"),
						},
						Elts: values,
					},
				},
			},
		},
	}
	return
}

func Obfuscate(file *ast.File, count int) *ast.File {
	keys, bootstrapVarName, bootstrapVarDecl := generateBootstrapVar()
	post := func(cursor *astutil.Cursor) bool {
		blockStmt, ok := cursor.Node().(*ast.BlockStmt)
		if !ok {
			return true
		}

		var funcType *ast.FuncType = nil

		switch f := cursor.Parent().(type) {
		case *ast.FuncLit:
			funcType = f.Type
		case *ast.FuncDecl:
			funcType = f.Type
		}
		newBlockStmt := mutateBlockStmt(blockStmt, keys, bootstrapVarName, funcType, count)
		if newBlockStmt != nil {
			cursor.Replace(newBlockStmt)
		}
		return true
	}

	file = astutil.Apply(file, nil, post).(*ast.File)
	file.Decls = append(file.Decls, bootstrapVarDecl)
	return file
}
