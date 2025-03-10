package ddlctlgo

import (
	"bytes"
	"go/ast"
	"go/token"
	"io"

	"github.com/hakadoriya/ddlctl/pkg/logs"
)

func dumpDDLSource(fset *token.FileSet, ddlSrc []*ddlSource) {
	for _, r := range ddlSrc {
		logs.Trace.Print("== ddlSource ================================================================================================================================")
		_, _ = io.WriteString(logs.Trace.LineWriter("r.CommentGroup.Text: "), r.CommentGroup.Text())
		logs.Trace.Print("-- CommentGroup --------------------------------------------------------------------------------------------------------------------------------")
		{
			commentGroupAST := bytes.NewBuffer(nil)
			ast.Fprint(commentGroupAST, fset, r.CommentGroup, ast.NotNilFilter)
			_, _ = logs.Trace.LineWriter("").Write(commentGroupAST.Bytes())
		}
		logs.Trace.Print("-- TypeSpec --------------------------------------------------------------------------------------------------------------------------------")
		{
			typeSpecAST := bytes.NewBuffer(nil)
			ast.Fprint(typeSpecAST, fset, r.TypeSpec, ast.NotNilFilter)
			_, _ = logs.Trace.LineWriter("").Write(typeSpecAST.Bytes())
		}
		logs.Trace.Print("-- StructType --------------------------------------------------------------------------------------------------------------------------------")
		{
			structTypeAST := bytes.NewBuffer(nil)
			ast.Fprint(structTypeAST, fset, r.StructType, ast.NotNilFilter)
			_, _ = logs.Trace.LineWriter("").Write(structTypeAST.Bytes())
		}
	}
}
