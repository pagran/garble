// Copyright (c) 2020, The Garble Authors.
// See LICENSE for licensing information.

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"strconv"
	"strings"

	ah "mvdan.cc/garble/internal/asthelper"
)

// updateMagicValue updates hardcoded value of hdr.magic
// when verifying header in symtab.go
func updateMagicValue(file *ast.File, magicValue uint32) {
	magicUpdated := false

	// Find `hdr.magic != 0xfffffff?` in symtab.go and update to random magicValue
	updateMagic := func(node ast.Node) bool {
		binExpr, ok := node.(*ast.BinaryExpr)
		if !ok || binExpr.Op != token.NEQ {
			return true
		}

		selectorExpr, ok := binExpr.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if ident, ok := selectorExpr.X.(*ast.Ident); !ok || ident.Name != "hdr" {
			return true
		}
		if selectorExpr.Sel.Name != "magic" {
			return true
		}

		if _, ok := binExpr.Y.(*ast.BasicLit); !ok {
			return true
		}
		binExpr.Y = &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatUint(uint64(magicValue), 10),
		}
		magicUpdated = true
		return false
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "moduledataverify1" {
			ast.Inspect(funcDecl, updateMagic)
			break
		}
	}

	if !magicUpdated {
		panic("magic value not updated")
	}
}

// updateEntryOffset adds xor encryption for funcInfo.entryoff
func updateEntryOffset(file *ast.File, entryOffKey uint32) {
	const entryOffRefCount = 2
	entryOffUpdated := 0

	updateEntryOff := func(cursor *astutil.Cursor) bool {
		selExpr, ok := cursor.Node().(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if selExpr.Sel.Name != "entryoff" {
			return true
		}

		cursor.Replace(&ast.BinaryExpr{
			X:  selExpr,
			Op: token.XOR,
			Y: &ast.BasicLit{
				Kind:  token.INT,
				Value: strconv.FormatUint(uint64(entryOffKey), 10),
			},
		})
		entryOffUpdated++
		return false
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && (funcDecl.Name.Name == "isInlined" || funcDecl.Name.Name == "entry") {
			astutil.Apply(funcDecl, nil, updateEntryOff)
		}
	}

	if entryOffUpdated != entryOffRefCount {
		panic(fmt.Sprintf("entryOffUpdated expected %d but found %d", entryOffRefCount, entryOffUpdated))
	}
}

// stripRuntime removes unnecessary code from the runtime,
// such as panic and fatal error printing, and code that
// prints trace/debug info of the runtime.
func stripRuntime(basename string, file *ast.File) {
	stripPrints := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		switch id.Name {
		case "print", "println":
			id.Name = "hidePrint"
			return false
		default:
			return true
		}
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		switch basename {
		case "error.go":
			// only used in panics
			switch funcDecl.Name.Name {
			case "printany", "printanycustomtype":
				funcDecl.Body.List = nil
			}
		case "mgcscavenge.go":
			// used in tracing the scavenger
			if funcDecl.Name.Name == "printScavTrace" {
				funcDecl.Body.List = nil
			}
		case "mprof.go":
			// remove all functions that print debug/tracing info
			// of the runtime
			if strings.HasPrefix(funcDecl.Name.Name, "trace") {
				funcDecl.Body.List = nil
			}
		case "panic.go":
			// used for printing panics
			switch funcDecl.Name.Name {
			case "preprintpanics", "printpanics":
				funcDecl.Body.List = nil
			}
		case "print.go":
			// only used in tracebacks
			if funcDecl.Name.Name == "hexdumpWords" {
				funcDecl.Body.List = nil
			}
		case "proc.go":
			// used in tracing the scheduler
			if funcDecl.Name.Name == "schedtrace" {
				funcDecl.Body.List = nil
			}
		case "runtime1.go":
			usesEnv := func(node ast.Node) bool {
				seen := false
				ast.Inspect(node, func(node ast.Node) bool {
					ident, ok := node.(*ast.Ident)
					if ok && ident.Name == "gogetenv" {
						seen = true
						return false
					}
					return true
				})
				return seen
			}
		filenames:
			switch funcDecl.Name.Name {
			case "parsedebugvars":
				// keep defaults for GODEBUG cgocheck and invalidptr,
				// remove code that reads GODEBUG via gogetenv
				for i, stmt := range funcDecl.Body.List {
					if usesEnv(stmt) {
						funcDecl.Body.List = funcDecl.Body.List[:i]
						break filenames
					}
				}
				panic("did not see any gogetenv call in parsedebugvars")
			case "setTraceback":
				// tracebacks are completely hidden, no
				// sense keeping this function
				funcDecl.Body.List = nil
			}
		case "traceback.go":
			// only used for printing tracebacks
			switch funcDecl.Name.Name {
			case "tracebackdefers", "printcreatedby", "printcreatedby1", "traceback", "tracebacktrap", "traceback1", "printAncestorTraceback",
				"printAncestorTracebackFuncInfo", "goroutineheader", "tracebackothers", "tracebackHexdump", "printCgoTraceback":
				funcDecl.Body.List = nil
			case "printOneCgoTraceback":
				funcDecl.Body = ah.BlockStmt(ah.ReturnStmt(ah.IntLit(0)))
			default:
				if strings.HasPrefix(funcDecl.Name.Name, "print") {
					funcDecl.Body.List = nil
				}
			}
		}

	}

	if basename == "print.go" {
		file.Decls = append(file.Decls, hidePrintDecl)
		return
	}

	// replace all 'print' and 'println' statements in
	// the runtime with an empty func, which will be
	// optimized out by the compiler
	ast.Inspect(file, stripPrints)
}

var hidePrintDecl = &ast.FuncDecl{
	Name: ast.NewIdent("hidePrint"),
	Type: &ast.FuncType{Params: &ast.FieldList{
		List: []*ast.Field{{
			Names: []*ast.Ident{{Name: "args"}},
			Type: &ast.Ellipsis{Elt: &ast.InterfaceType{
				Methods: &ast.FieldList{},
			}},
		}},
	}},
	Body: &ast.BlockStmt{},
}
