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
	case *parser.ArrayType:
		var buf strings.Builder
		for _, indexType := range dt.IndexTypes {
			buf.WriteString("[")
			srt, ok := indexType.(*parser.SubrangeType)
			if ok {
				buf.WriteString(fmt.Sprintf("%d", srt.UpperBound-srt.LowerBound+1))
			} // TODO: handle other index types.
			buf.WriteString("]")
		}
		buf.WriteString(toGoType(dt.ElementType))
		return buf.String()
	case *parser.SubrangeType:
		return "int" // Go doesn't have subrange types, so that's the closest we can translate them to.
	case *parser.EnumType:
		return "int" // Go doesn't have enum types, so we just define it as an alias to int, and declare constants and a string conversion method.
	case *parser.SetType:
		return fmt.Sprintf("system.SetType[%s]", toGoType(dt.ElementType))
	case *parser.FileType:
		// TODO: implement
	case *parser.ProcedureType:
		var buf strings.Builder
		buf.WriteString("func(")
		for idx, param := range dt.FormalParams {
			if idx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(toGoType(param.Type))
		}
		buf.WriteString(")")
		return buf.String()
	case *parser.FunctionType:
		var buf strings.Builder
		buf.WriteString("func(")
		for idx, param := range dt.FormalParams {
			if idx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(toGoType(param.Type))
		}
		buf.WriteString(") ")
		buf.WriteString(toGoType(dt.ReturnType))
		return buf.String()
	}
	return fmt.Sprintf("bug: unhandled type %T", typ)
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

var operatorMapping = map[string]string{
	"=":   "==",
	"<>":  "!=",
	"<":   "<",
	">":   ">",
	"<=":  "<=",
	">=":  ">=",
	"+":   "+",
	"-":   "-",
	"*":   "*",
	"/":   "/",
	"div": "/",
	"mod": "%",
	"and": "&&",
	"or":  "||",
}

func translateOperator(op string) string {
	newOp, ok := operatorMapping[op]
	if !ok {
		return fmt.Sprintf("BUG(unknown operator %q)", op)
	}
	return newOp
}

func toSetSimpleExpr(e *parser.SimpleExpr) string {
	var buf strings.Builder

	buf.WriteString(e.Sign) // TODO: this makes no sense, so how should we handle this?

	buf.WriteString(toExpr(e.First))
	for _, next := range e.Next {
		switch next.Operator {
		case parser.OperatorAdd:
			buf.WriteString(".Union(")
			buf.WriteString(toExpr(next.Term))
			buf.WriteString(")")
		case parser.OperatorSubtract:
			buf.WriteString(".Difference(")
			buf.WriteString(toExpr(next.Term))
			buf.WriteString(")")
		default:
			fmt.Fprintf(&buf, "BUG: unsupported operator %s", string(next.Operator))
		}
	}

	return buf.String()
}

func toSetTermExpr(e *parser.TermExpr) string {
	var buf strings.Builder

	buf.WriteString(toExpr(e.First))

	for _, next := range e.Next {
		switch next.Operator {
		case parser.OperatorMultiply:
			buf.WriteString(".Intersection(")
			buf.WriteString(toExpr(next.Factor))
			buf.WriteString(")")
		default:
			fmt.Fprintf(&buf, "BUG: unsupported operator %s", string(next.Operator))
		}
	}

	return buf.String()
}

func toExpr(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.RelationalExpr:
		if e.Operator == parser.OpIn {
			return toExpr(e.Right) + ".In(" + toExpr(e.Left) + ")"
		}
		return toExpr(e.Left) + " " + translateOperator(string(e.Operator)) + " " + toExpr(e.Right)
	case *parser.SimpleExpr:
		if _, isSetType := e.First.Type().(*parser.SetType); isSetType {
			return toSetSimpleExpr(e)
		}
		var buf strings.Builder
		buf.WriteString(e.Sign)
		buf.WriteString(toExpr(e.First))
		for _, next := range e.Next {
			buf.WriteString(translateOperator(string(next.Operator)))
			buf.WriteString(toExpr(next.Term))
		}
		return buf.String()
	case *parser.TermExpr:
		if _, isSetType := e.First.Type().(*parser.SetType); isSetType {
			return toSetTermExpr(e)
		}
		var buf strings.Builder
		buf.WriteString(toExpr(e.First))
		for _, next := range e.Next {
			buf.WriteString(translateOperator(string(next.Operator)))
			buf.WriteString(toExpr(next.Factor))
		}
		return buf.String()
	case *parser.ConstantExpr:
		return e.Name
	case *parser.VariableExpr:
		str := e.Name
		if e.VarDecl != nil {
			decl := e.VarDecl
			for decl.BelongsTo != "" {
				str = decl.BelongsTo + "." + str
				decl = decl.BelongsToDecl
			}
			return str
		}
		if e.IsReturnValue {
			str = str + "_"
		}
		return str
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
		var buf strings.Builder
		buf.WriteString("system.Set[" + toGoType(e.Type().(*parser.SetType).ElementType) + "](")
		for idx, expr := range e.Elements {
			if idx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(toExpr(expr))
		}
		buf.WriteString(")")
		return buf.String()
	case *parser.SubExpr:
		return "(" + toExpr(e.Expr) + ")"
	case *parser.IndexedVariableExpr:
		var buf strings.Builder
		buf.WriteString(toExpr(e.Expr))
		for idx, idxExpr := range e.IndexExprs {
			buf.WriteString("[")
			buf.WriteString(toExpr(idxExpr))
			if srt, ok := e.Expr.Type().(*parser.ArrayType).IndexTypes[idx].(*parser.SubrangeType); ok && srt.LowerBound != 0 {
				buf.WriteString("-(")
				buf.WriteString(fmt.Sprint(srt.LowerBound))
				buf.WriteString(")")
			}
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
		return "(*" + toExpr(e.Expr) + ")"
	case *parser.FormatExpr:
		// TODO: implement full formatting
		return toExpr(e.Expr)
	case *parser.CharExpr:
		return fmt.Sprintf("'%c'", e.Value)
	default:
		return fmt.Sprintf("bug: invalid expression type %T", expr)
	}
}

func filterEnumTypes(typeDefs []*parser.TypeDefinition) (enumTypes []*parser.TypeDefinition) {
	for _, typeDef := range typeDefs {
		if _, ok := typeDef.Type.(*parser.EnumType); ok {
			enumTypes = append(enumTypes, typeDef)
		}
	}
	return enumTypes
}

func generateEnumConstants(typeDef *parser.TypeDefinition) string {
	var buf strings.Builder

	buf.WriteString("const (\n")
	for identIdx, ident := range typeDef.Type.(*parser.EnumType).Identifiers {
		fmt.Fprintf(&buf, "	%s %s = %d\n", ident, typeDef.Name, identIdx)
	}
	buf.WriteString(")\n")

	return buf.String()
}
