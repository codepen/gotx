package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "gotx",
	Doc:  "reports references of a transaction's receiver type from inside the transaction",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		insideTransaction := false
		var transactionReceiverType types.Type
		var begTransactionPos token.Pos
		var endTransactionPos token.Pos

		ast.Inspect(file, func(n ast.Node) bool {
			if !insideTransaction {
				ce, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				se, ok := ce.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				if !(se.Sel.Name == "InsideTx" || se.Sel.Name == "InsideTransaction") {
					return true
				}

				insideTransaction = true
				transactionReceiverType = pass.TypesInfo.TypeOf(se.X)
				begTransactionPos = ce.Lparen
				endTransactionPos = ce.Rparen

				return true
			}

			if n == nil {
				return true
			}

			if n.Pos() < begTransactionPos {
				// Transaction block hasn't started yet
				return true
			}

			if n.Pos() >= endTransactionPos {
				// Transaction block ended
				insideTransaction = false
				return true
			}

			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}

			if pass.TypesInfo.TypeOf(ident) == transactionReceiverType {
				pass.Reportf(ident.Pos(), "transaction receiver's type used inside transaction - only the transaction type should be used %q",
					render(pass.Fset, ident))
			}

			return true
		})
	}

	return nil, nil
}

func render(fset *token.FileSet, x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}
