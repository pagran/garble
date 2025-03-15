// Copyright (c) 2020, The Garble Authors.
// See LICENSE for licensing information.

package literals

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"
	mathrand "math/rand"
	ah "mvdan.cc/garble/internal/asthelper"
)

type externalKey struct {
	idx   int
	typ   string
	value uint64
}

func (k externalKey) IsJunk() bool {
	return k.idx == -1
}

func (k externalKey) Byte() byte {
	return byte(k.value)
}

func (k externalKey) Var() ast.Expr {
	if k.IsJunk() {
		panic("externalKey.Var() called on junk externalKey")
	}
	return ah.CallExprByName("byte", k.Name())
}

func (k externalKey) Param() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{k.Name()},
		Type:  k.Type(),
	}
}

func (k externalKey) Arg() ast.Expr {
	return &ast.BasicLit{
		Kind:  token.INT,
		Value: fmt.Sprintln(k.value),
	}
}

func (k externalKey) Type() *ast.Ident {
	return ast.NewIdent(k.typ)
}

func (k externalKey) Name() *ast.Ident {
	if k.idx == -1 {
		return ast.NewIdent("_")
	}
	return ast.NewIdent(fmt.Sprintf("_ek_%d", k.idx))
}

// obfuscator takes a byte slice and converts it to a ast.BlockStmt
type obfuscator interface {
	obfuscate(obfRand *mathrand.Rand, data []byte) (*ast.BlockStmt, []externalKey)
}

var (
	simpleObfuscator = simple{}

	// Obfuscators contains all types which implement the obfuscator Interface
	Obfuscators = []obfuscator{
		simpleObfuscator,
		//swap{},
		//split{},
		//shuffle{},
		//seed{},
	}

	// LinearTimeObfuscators contains all types which implement the obfuscator Interface and can safely be used on large literals
	LinearTimeObfuscators = []obfuscator{
		simpleObfuscator,
	}

	TestObfuscator         string
	testPkgToObfuscatorMap map[string]obfuscator
)

func genRandIntSlice(obfRand *mathrand.Rand, max, count int) []int {
	indexes := make([]int, count)
	for i := range count {
		indexes[i] = obfRand.Intn(max)
	}
	return indexes
}

func randOperator(obfRand *mathrand.Rand) token.Token {
	operatorTokens := [...]token.Token{token.XOR, token.ADD, token.SUB}
	return operatorTokens[obfRand.Intn(len(operatorTokens))]
}

func evalOperator(t token.Token, x, y byte) byte {
	switch t {
	case token.XOR:
		return x ^ y
	case token.ADD:
		return x + y
	case token.SUB:
		return x - y
	default:
		panic(fmt.Sprintf("unknown operator: %s", t))
	}
}

func operatorToReversedBinaryExpr(t token.Token, x, y ast.Expr) *ast.BinaryExpr {
	expr := &ast.BinaryExpr{X: x, Y: y}

	switch t {
	case token.XOR:
		expr.Op = token.XOR
	case token.ADD:
		expr.Op = token.SUB
	case token.SUB:
		expr.Op = token.ADD
	default:
		panic(fmt.Sprintf("unknown operator: %s", t))
	}

	return expr
}

const maxExternalKeyCount = 8

var externalKeyRanges = []struct {
	typ string
	max uint64
}{
	{"uint8", math.MaxUint8},
	{"uint16", math.MaxUint16},
	{"uint32", math.MaxUint32},
	{"uint64", math.MaxUint64},
}

func randExternalKey(obfRand *mathrand.Rand, idx int) externalKey {
	r := externalKeyRanges[obfRand.Intn(len(externalKeyRanges))]
	return externalKey{
		idx:   idx,
		typ:   r.typ,
		value: obfRand.Uint64() & r.max,
	}
}

func randExternalKeys(obfRand *mathrand.Rand, maxCount int) []externalKey {
	if maxCount > maxExternalKeyCount {
		maxCount = maxExternalKeyCount
	}
	count := 1 + obfRand.Intn(maxCount)

	keys := make([]externalKey, count)
	for i := 0; i < count; i++ {
		keys[i] = randExternalKey(obfRand, i)
	}

	return keys
}

func addTrashExternalKeys(obfRand *mathrand.Rand, keys []externalKey) []externalKey {
	if len(keys) >= maxExternalKeyCount {
		return keys
	}

	trashCount := maxExternalKeyCount - len(keys)
	if trashCount > 1 {
		trashCount = 1 + obfRand.Intn(trashCount)
	}
	for i := 0; i < trashCount; i++ {
		keys = append(keys, randExternalKey(obfRand, -1))
	}

	obfRand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	return keys
}

func externalKeysToParams(obfRand *mathrand.Rand, keys []externalKey) (params *ast.FieldList, args []ast.Expr) {
	params = &ast.FieldList{}
	for _, key := range addTrashExternalKeys(obfRand, keys) {
		params.List = append(params.List, key.Param())
		args = append(args, key.Arg())
	}
	return
}

type obfRand struct {
	*mathrand.Rand
	testObfuscator obfuscator
}

func (r *obfRand) nextObfuscator() obfuscator {
	if r.testObfuscator != nil {
		return r.testObfuscator
	}
	return Obfuscators[r.Intn(len(Obfuscators))]
}

func (r *obfRand) nextLinearTimeObfuscator() obfuscator {
	if r.testObfuscator != nil {
		return r.testObfuscator
	}
	return Obfuscators[r.Intn(len(LinearTimeObfuscators))]
}

func newObfRand(rand *mathrand.Rand, file *ast.File) *obfRand {
	testObf := testPkgToObfuscatorMap[file.Name.Name]
	return &obfRand{rand, testObf}
}
