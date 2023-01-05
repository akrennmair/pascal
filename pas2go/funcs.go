package pas2go

import (
	"fmt"
	"strings"

	"github.com/akrennmair/pascal/parser"
)

func toGoType(typ parser.DataType) string {
	switch dt := typ.(type) {
	case *parser.IntegerType:
		return "int"
	case *parser.BooleanType:
		return "bool"
	case *parser.RealType:
		return "float64"
	case *parser.RecordType:
		if name := typ.TypeName(); name != "" {
			return name
		}
		return recordTypeToGoType(dt)
	case *parser.CharType:
		return "byte"
	case *parser.StringType:
		return "string"
	case *parser.PointerType:
		if name := typ.TypeName(); name != "" {
			return name
		}
		if dt.Name != "" {
			return "*" + dt.Name
		}
		return "*" + toGoType(dt.Type_)
	default:
		_ = dt
		return fmt.Sprintf("bug: unhandled type %T", typ)
	}
}

func sortTypeDefs(typeDefs []*parser.TypeDefinition) []*parser.TypeDefinition {
	var (
		newTypeList   []*parser.TypeDefinition
		typesToAppend []*parser.TypeDefinition
	)

	for _, typeDef := range typeDefs {
		if pt, ok := typeDef.Type.(*parser.PointerType); ok {
			if pt.Name != "" {
				typesToAppend = append(typesToAppend, typeDef)
				continue
			}
		}
		newTypeList = append(newTypeList, typeDef)
	}

	return append(newTypeList, typesToAppend...)
}

func recordTypeToGoType(rec *parser.RecordType) string {
	var buf strings.Builder

	buf.WriteString("struct {\n")

	for _, field := range rec.Fields {
		buf.WriteString("	")
		buf.WriteString(field.Identifier)
		buf.WriteString(" ")
		buf.WriteString(toGoType(field.Type))
		buf.WriteString("\n")
	}

	if rec.VariantField != nil {
		if rec.VariantField.TagField != "" && rec.VariantField.Type != nil {
			buf.WriteString("    ")
			buf.WriteString(rec.VariantField.TagField)
			buf.WriteString(" ")
			buf.WriteString(rec.VariantField.Type.TypeName())
			buf.WriteString(" `pas2go:\"tagfield\"`")
			buf.WriteString("\n")
		}
		for _, variant := range rec.VariantField.Variants {
			var caseLabels []string
			for _, l := range variant.CaseLabels {
				caseLabels = append(caseLabels, l.String())
			}
			for _, field := range variant.Fields.Fields {
				buf.WriteString("	")
				buf.WriteString(field.Identifier)
				buf.WriteString(" ")
				buf.WriteString(toGoType(field.Type))
				buf.WriteString(fmt.Sprintf(" `pas2go:\"caselabels,%s\"`", strings.Join(caseLabels, ",")))
				buf.WriteString("\n")
			}
		}
	}

	buf.WriteString("}")

	return buf.String()
}

func constantLiteral(cl parser.ConstantLiteral) string {
	switch lit := cl.(type) {
	case *parser.IntegerLiteral:
		return fmt.Sprint(lit.Value)
	case *parser.StringLiteral:
		return fmt.Sprintf("%q", lit.Value)
	case *parser.RealLiteral:
		sign := ""
		if lit.Minus {
			sign = "-"
		}
		return fmt.Sprintf("%s%s.%se%d", sign, lit.BeforeComma, lit.AfterComma, lit.ScaleFactor)
	case *parser.EnumValueLiteral:
		return lit.Symbol
	default:
		return fmt.Sprintf("bug: unhandled constant literal type %T", cl)
	}
}

func formalParams(params []*parser.FormalParameter) string {
	var buf strings.Builder

	for idx, param := range params {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(param.Name)
		buf.WriteString(" ")
		buf.WriteString(toGoType(param.Type))
	}

	return buf.String()
}

func actualParams(params []parser.Expression) string {
	var buf strings.Builder

	buf.WriteString("(")

	for idx, param := range params {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(toExpr(param))
	}

	buf.WriteString(")")

	return buf.String()
}

func toExpr(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.RelationalExpr:
		return toExpr(e.Left) + " " + string(e.Operator) + " " + toExpr(e.Right)
	case *parser.SimpleExpr:
		var buf strings.Builder
		buf.WriteString(e.Sign)
		buf.WriteString(toExpr(e.First))
		for _, next := range e.Next {
			buf.WriteString(string(next.Operator))
			buf.WriteString(toExpr(next.Term))
		}
		return buf.String()
	case *parser.TermExpr:
		var buf strings.Builder
		buf.WriteString(toExpr(e.First))
		for _, next := range e.Next {
			buf.WriteString(string(next.Operator))
			buf.WriteString(toExpr(next.Factor))
		}
		return buf.String()
	case *parser.ConstantExpr:
		return e.Name
	case *parser.VariableExpr:
		if e.Decl != nil && e.Decl.BelongsTo != "" {
			return e.Decl.BelongsTo + "." + e.Name
		}
		return e.Name
	case *parser.IntegerExpr:
		return fmt.Sprint(e.Value)
	case *parser.RealExpr:
		sign := ""
		if e.Minus {
			sign = "-"
		}
		return fmt.Sprintf("%s%s.%se%d", sign, e.BeforeComma, e.AfterComma, e.ScaleFactor)
	case *parser.StringExpr:
		return fmt.Sprintf("%q", e.Value)
	case *parser.NilExpr:
		return "nil"
	case *parser.NotExpr:
		return "!" + toExpr(e.Expr)
	case *parser.SetExpr:
		// TODO: implement
		return "TODO: implement set expression"
	case *parser.SubExpr:
		return "(" + toExpr(e.Expr) + ")"
	case *parser.IndexedVariableExpr:
		var buf strings.Builder
		buf.WriteString(toExpr(e.Expr))
		for _, idxExpr := range e.IndexExprs {
			buf.WriteString("[")
			buf.WriteString(toExpr(idxExpr))
			buf.WriteString("]")
		}
		return buf.String()
	case *parser.FunctionCallExpr:
		return e.Name + actualParams(e.ActualParams)
	case *parser.FieldDesignatorExpr:
		return toExpr(e.Expr) + "." + e.Field
	case *parser.EnumValueExpr:
		return e.Name
	case *parser.DerefExpr:
		return "*(" + toExpr(e.Expr) + ")"
	case *parser.FormatExpr:
		// TODO: implement full formatting
		return toExpr(e.Expr)
	case *parser.CharExpr:
		return fmt.Sprintf("'%c'", e.Value)
	default:
		return fmt.Sprintf("bug: invalid expression type %T", expr)
	}
}
