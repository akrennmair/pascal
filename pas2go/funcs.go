package pas2go

import (
	"fmt"
	"strings"

	"github.com/akrennmair/pascal/parser"
)

func toGoTypeDef(typeDef *parser.TypeDefinition) string {
	var buf strings.Builder

	buf.WriteString(typeDef.Name)
	buf.WriteString(" ")
	buf.WriteString(toGoTypeExcludeTypeName(typeDef.Type, typeDef.Name))

	return buf.String()
}

func toGoType(typ parser.DataType) string {
	return toGoTypeExcludeTypeName(typ, "")
}

func toGoTypeExcludeTypeName(typ parser.DataType, excludeTypeName string) string {
	switch dt := typ.(type) {
	case *parser.IntegerType:
		return "int"
	case *parser.RealType:
		return "float64"
	case *parser.RecordType:
		if name := typ.TypeName(); name != "" {
			return name
		}
		return recordTypeToGoType(dt)
	case *parser.StringType:
		return "string"
	case *parser.PointerType:
		if name := typ.TypeName(); name != "" && name != excludeTypeName {
			return name
		}

		if dt.TargetName != "" {
			return "*" + dt.TargetName
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
		if name := typ.TypeName(); name != "" && name != excludeTypeName {
			return name
		}

		if parser.IsCharType(typ) {
			return "byte"
		}

		return "int" // Go doesn't have subrange types, so that's the closest we can translate them to.
	case *parser.EnumType:
		if parser.IsBooleanType(typ) {
			return "bool"
		}

		if name := typ.TypeName(); name != "" && name != excludeTypeName {
			return name
		}

		return "int" // Go doesn't have enum types, so we just define it as an alias to int, and declare constants and a string conversion method.
	case *parser.SetType:
		return fmt.Sprintf("system.SetType[%s]", toGoType(dt.ElementType))
	case *parser.FileType:
		return fmt.Sprintf("system.FileType[%s]", toGoType(dt.ElementType))
	case *parser.ProcedureType:
		var buf strings.Builder
		buf.WriteString("func(")
		for idx, param := range dt.FormalParams {
			if idx > 0 {
				buf.WriteString(", ")
			}
			if param.VariableParameter {
				buf.WriteString("*")
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
			if param.VariableParameter {
				buf.WriteString("*")
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
			if pt.TargetName != "" {
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
			buf.WriteString(toGoType(rec.VariantField.Type))
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
		if lit.Value < 0 {
			return fmt.Sprintf("(%d)", lit.Value)
		}
		return fmt.Sprint(lit.Value)
	case *parser.StringLiteral:
		return fmt.Sprintf("%q", lit.Value)
	case *parser.RealLiteral:
		sign := ""
		if lit.Minus {
			sign = "-"
		}
		realStr := fmt.Sprintf("%s%s.%se%d", sign, lit.BeforeComma, lit.AfterComma, lit.ScaleFactor)
		if sign != "" {
			return "(" + realStr + ")"
		}
		return realStr
	case *parser.EnumValueLiteral:
		return lit.Symbol
	case *parser.CharLiteral:
		if lit.Value == '\'' {
			return `'\''`
		}
		return fmt.Sprintf("'%c'", lit.Value)
	default:
		return fmt.Sprintf("bug: unhandled constant literal type %T", cl)
	}
}

func constantLiteralList(labels []parser.ConstantLiteral) string {
	var buf strings.Builder

	for idx, l := range labels {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(constantLiteral(l))
	}

	return buf.String()
}

func formalParams(params []*parser.FormalParameter) string {
	var buf strings.Builder

	for idx, param := range params {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(param.Name)
		buf.WriteString(" ")
		if param.VariableParameter {
			buf.WriteString("*")
		}
		buf.WriteString(toGoType(param.Type))
	}

	return buf.String()
}

func actualParams(params []parser.Expression, formalParams []*parser.FormalParameter) string {
	var buf strings.Builder

	buf.WriteString("(")

	for idx, param := range params {
		if idx > 0 {
			buf.WriteString(", ")
		}
		if formalParams != nil && formalParams[idx].VariableParameter {
			buf.WriteString("&")
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

func toSetRelationalExpr(e *parser.RelationalExpr) string {
	var buf strings.Builder

	switch e.Operator {
	case parser.OpEqual:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".Equals(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	case parser.OpNotEqual:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".NotEquals(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	case parser.OpLess:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".Less(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	case parser.OpLessEqual:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".LessEqual(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	case parser.OpGreater:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".Greater(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	case parser.OpGreaterEqual:
		buf.WriteString(toExpr(e.Left))
		buf.WriteString(".GreaterEqual(")
		buf.WriteString(toExpr(e.Right))
		buf.WriteString(")")
	default:
		fmt.Fprintf(&buf, "BUG: unsupported set relational operator %s", string(e.Operator))
	}

	return buf.String()
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

func findTypeConversion(leftExpr parser.Expression, rightExpr parser.Expression) string {
	_, leftIsReal := leftExpr.Type().(*parser.RealType)
	_, rightIsInt := rightExpr.Type().(*parser.IntegerType)
	_, rightIsIntLiteral := rightExpr.(*parser.IntegerExpr)

	if leftIsReal && rightIsInt && !rightIsIntLiteral {
		return "float64"
	}

	return ""
}

func findTypeConversion2(leftType parser.DataType, rightExpr parser.Expression) string {
	_, leftIsReal := leftType.(*parser.RealType)
	_, rightIsInt := rightExpr.Type().(*parser.IntegerType)
	_, rightIsIntLiteral := rightExpr.(*parser.IntegerExpr)

	if leftIsReal && rightIsInt && !rightIsIntLiteral {
		return "float64"
	}

	return ""
}

func findLeftTypeConversion(leftExpr parser.Expression, rightExpr parser.Expression) (leftType parser.DataType, typeConv string) {
	_, rightIsReal := rightExpr.Type().(*parser.RealType)
	_, leftIsInt := leftExpr.Type().(*parser.IntegerType)
	_, leftIsIntLiteral := leftExpr.(*parser.IntegerExpr)

	if rightIsReal && leftIsInt && !leftIsIntLiteral {
		return rightExpr.Type(), "float64"
	}

	return leftExpr.Type(), ""
}

func applyTypeConversion(newType string, expr string) string {
	var buf strings.Builder
	if newType != "" {
		buf.WriteString(newType)
		buf.WriteString("(")
	}
	buf.WriteString(expr)
	if newType != "" {
		buf.WriteString(")")
	}
	return buf.String()
}

func toExpr(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.RelationalExpr:
		leftExpr := toExpr(e.Left)
		rightExpr := toExpr(e.Right)
		if e.Operator == parser.OpIn {
			return rightExpr + ".In(" + leftExpr + ")"
		}
		if _, isSetType := e.Left.Type().(*parser.SetType); isSetType {
			return toSetRelationalExpr(e)
		}
		if parser.IsBooleanType(e.Left.Type()) && parser.IsBooleanType(e.Right.Type()) {
			if e.Operator == parser.OpGreater || e.Operator == parser.OpGreaterEqual || e.Operator == parser.OpLess || e.Operator == parser.OpLessEqual {
				leftExpr = "system.BoolOrd(" + leftExpr + ")"
				rightExpr = "system.BoolOrd(" + rightExpr + ")"
			}
		} else if isStringish(e.Left.Type()) && isStringish(e.Right.Type()) {
			if isCharArray(e.Left.Type()) {
				leftExpr = fmt.Sprintf("string(%s[:])", leftExpr)
			}
			if isCharArray(e.Right.Type()) {
				rightExpr = fmt.Sprintf("string(%s[:])", rightExpr)
			}
		}
		return leftExpr + " " + translateOperator(string(e.Operator)) + " " + rightExpr
	case *parser.SimpleExpr:
		if _, isSetType := e.First.Type().(*parser.SetType); isSetType {
			return toSetSimpleExpr(e)
		}
		var buf strings.Builder
		buf.WriteString(e.Sign)
		if len(e.Next) > 0 {
			leftType, typeConv := findLeftTypeConversion(e.First, e.Next[0].Term)
			buf.WriteString(applyTypeConversion(typeConv, toExpr(e.First)))
			for _, next := range e.Next {
				typeConv := findTypeConversion2(leftType, next.Term)
				buf.WriteString(translateOperator(string(next.Operator)))
				buf.WriteString(applyTypeConversion(typeConv, toExpr(next.Term)))
			}
		} else {
			buf.WriteString(toExpr(e.First))
			for _, next := range e.Next {
				typeConv := findTypeConversion(e.First, next.Term)
				buf.WriteString(translateOperator(string(next.Operator)))
				buf.WriteString(applyTypeConversion(typeConv, toExpr(next.Term)))
			}
		}

		return buf.String()
	case *parser.TermExpr:
		if _, isSetType := e.First.Type().(*parser.SetType); isSetType {
			return toSetTermExpr(e)
		}
		var buf strings.Builder
		if len(e.Next) > 0 {
			leftType, typeConv := findLeftTypeConversion(e.First, e.Next[0].Factor)
			buf.WriteString(applyTypeConversion(typeConv, toExpr(e.First)))
			for _, next := range e.Next {
				typeConv := findTypeConversion2(leftType, next.Factor)
				buf.WriteString(translateOperator(string(next.Operator)))
				buf.WriteString(applyTypeConversion(typeConv, toExpr(next.Factor)))
			}
		} else {
			buf.WriteString(toExpr(e.First))
			for _, next := range e.Next {
				typeConv := findTypeConversion(e.First, next.Factor)
				buf.WriteString(translateOperator(string(next.Operator)))
				buf.WriteString(applyTypeConversion(typeConv, toExpr(next.Factor)))
			}
		}

		return buf.String()
	case *parser.ConstantExpr:
		if str, ok := getBuiltinConstant(e.Name); ok {
			return str
		}
		return e.Name
	case *parser.VariableExpr:
		return toVariableExpr(e)
	case *parser.IntegerExpr:
		if e.Value < 0 {
			return fmt.Sprintf("(%d)", e.Value)
		}
		return fmt.Sprint(e.Value)
	case *parser.RealExpr:
		sign := ""
		if e.Minus {
			sign = "-"
		}
		realStr := fmt.Sprintf("%s%s.%se%d", sign, e.BeforeComma, e.AfterComma, e.ScaleFactor)
		if sign != "" {
			realStr = "(" + realStr + ")"
		}
		return realStr
	case *parser.StringExpr:
		return fmt.Sprintf("%q", e.Value)
	case *parser.NilExpr:
		return "nil"
	case *parser.NotExpr:
		return "!" + toExpr(e.Expr)
	case *parser.SetExpr:
		var buf strings.Builder
		buf.WriteString("system.Set")

		if elemTyp := e.Type().(*parser.SetType).ElementType; elemTyp != nil {
			buf.WriteString("[")
			buf.WriteString(toGoType(elemTyp))
			buf.WriteString("]")
		}
		buf.WriteString("(")
		for idx, expr := range e.Elements {
			if idx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(toExpr(expr))
		}
		buf.WriteString(")")
		return buf.String()
	case *parser.RangeExpr:
		var buf strings.Builder
		buf.WriteString(fmt.Sprintf("system.Range[%s](", toGoType(e.LowerBound.Type())))
		buf.WriteString(toExpr(e.LowerBound))
		buf.WriteString(", ")
		buf.WriteString(toExpr(e.UpperBound))
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
		return toFunctionCallExpr(e)
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
		if e.Value == '\'' {
			return `'\''`
		}
		return fmt.Sprintf("'%c'", e.Value)
	default:
		return fmt.Sprintf("bug: invalid expression type %T", expr)
	}
}

func toVariableExpr(e *parser.VariableExpr) string {
	if e.IsReturnValue {
		return e.Name + "_"
	}

	if e.ParamDecl != nil && e.ParamDecl.VariableParameter {
		return "(*" + e.Name + ")"
	}

	str := e.Name
	varDecl := e.VarDecl
	if varDecl != nil && varDecl.IsRecordField {
		str = toExpr(varDecl.BelongsToExpr) + "." + str // TODO: add fix for when expression refers to variable parameter.
	}

	return str
}

func toFunctionCallExpr(e *parser.FunctionCallExpr) string {
	switch e.Name {
	case "abs":
		switch e.ActualParams[0].Type().(type) {
		case *parser.IntegerType:
			return "system.AbsInt" + actualParams(e.ActualParams, e.FormalParams)
		case *parser.RealType:
			return "system.AbsReal" + actualParams(e.ActualParams, e.FormalParams)
		case *parser.SubrangeType:
			return "system.AbsInt(" + toExpr(e.ActualParams[0]) + ")"
		}
	case "arctan":
		return "system.Arctan" + actualParams(e.ActualParams, e.FormalParams)
	case "cos":
		return "system.Cos" + actualParams(e.ActualParams, e.FormalParams)
	case "exp":
		return "system.Exp" + actualParams(e.ActualParams, e.FormalParams)
	case "frac":
		return "system.Frac" + actualParams(e.ActualParams, e.FormalParams)
	case "int":
		return "system.Int" + actualParams(e.ActualParams, e.FormalParams)
	case "ln":
		return "system.Exp" + actualParams(e.ActualParams, e.FormalParams)
	case "pi":
		return "system.Pi" + actualParams(e.ActualParams, e.FormalParams)
	case "sin":
		return "system.Sin" + actualParams(e.ActualParams, e.FormalParams)
	case "sqr":
		switch e.ActualParams[0].Type().(type) {
		case *parser.IntegerType:
			return "system.SqrInt" + actualParams(e.ActualParams, e.FormalParams)
		case *parser.RealType:
			return "system.Sqr" + actualParams(e.ActualParams, e.FormalParams)
		default:
			return fmt.Sprintf("BUG: unexpected type %s", e.ActualParams[0].Type().TypeString())
		}
	case "sqrt":
		return "system.Sqrt" + actualParams(e.ActualParams, e.FormalParams)
	case "trunc":
		return "system.Trunc" + actualParams(e.ActualParams, e.FormalParams)
	case "round":
		return "system.Round" + actualParams(e.ActualParams, e.FormalParams)
	case "chr":
		return "system.Chr" + actualParams(e.ActualParams, e.FormalParams)
	case "odd":
		return "system.Odd" + actualParams(e.ActualParams, e.FormalParams)
	case "ord":
		param := e.ActualParams[0]
		if parser.IsBooleanType(param.Type()) {
			return "system.BoolOrd(" + toExpr(param) + ")"
		} else if se, ok := param.(*parser.StringExpr); ok {
			param = &parser.CharExpr{
				Value: se.Value[0],
			}
		} else if ce, ok := param.(*parser.ConstantExpr); ok {
			param = ce
		}
		return "int(" + toExpr(param) + ")"
	case "succ":
		if parser.IsBooleanType(e.ActualParams[0].Type()) {
			return "system.BoolSucc(" + toExpr(e.ActualParams[0]) + ")"
		}
		return "(" + toExpr(e.ActualParams[0]) + " + 1)"
	case "pred":
		if parser.IsBooleanType(e.ActualParams[0].Type()) {
			return "system.BoolPred(" + toExpr(e.ActualParams[0]) + ")"
		}
		return "(" + toExpr(e.ActualParams[0]) + " - 1)"
	}

	return e.Name + actualParams(e.ActualParams, e.FormalParams)
}

func generateEnumValue(enumValue *parser.EnumValue) string {
	var buf strings.Builder

	buf.WriteString(enumValue.Name)
	buf.WriteString(" ")

	if name := enumValue.Type.TypeName(); name != "" {
		buf.WriteString(name)
		buf.WriteString(" ")
	}

	buf.WriteString("= ")
	buf.WriteString(fmt.Sprintf("%d", enumValue.Value))

	return buf.String()
}

func isBuiltinProcedure(name string) bool {
	return parser.FindBuiltinProcedure(name) != nil
}

func getBuiltinConstant(ident string) (string, bool) {
	switch ident {
	case "maxint":
		return "system.MaxInt", true
	}
	return "", false
}

func generateBuiltinProcedure(stmt *parser.ProcedureCallStatement) string {
	switch stmt.Name {
	case "new":
		typ := stmt.ActualParams[0].Type().(*parser.PointerType).Type_
		return toExpr(stmt.ActualParams[0]) + " = new(" + toGoType(typ) + ")"
	case "dispose":
		return toExpr(stmt.ActualParams[0]) + " = nil"
	case "read":
		return "system.Read" + toPointerParamList(stmt.ActualParams)
	case "readln":
		return "system.Readln" + toPointerParamList(stmt.ActualParams)
	case "inc":
		switch len(stmt.ActualParams) {
		case 1:
			return toExpr(stmt.ActualParams[0]) + "++"
		case 2:
			return toExpr(stmt.ActualParams[0]) + " += " + toExpr(stmt.ActualParams[1])
		}
	case "dec":
		switch len(stmt.ActualParams) {
		case 1:
			return toExpr(stmt.ActualParams[0]) + "--"
		case 2:
			return toExpr(stmt.ActualParams[0]) + " -= " + toExpr(stmt.ActualParams[1])
		}
	case "rewrite", "reset", "unpack", "pack", "get", "put":
		return fmt.Sprintf("/* TODO: %s%s */", stmt.Name, actualParams(stmt.ActualParams, nil))
	}
	return "BUG: missing builtin procedure " + stmt.Name
}

func toPointerParamList(params []parser.Expression) string {
	var buf strings.Builder

	buf.WriteString("(")

	for idx, param := range params {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("&")
		buf.WriteString(toExpr(param))
	}

	buf.WriteString(")")

	return buf.String()
}

func isCharArray(typ parser.DataType) bool {
	arrType, ok := typ.(*parser.ArrayType)
	return ok && parser.IsCharType(arrType.ElementType)
}

func isString(typ parser.DataType) bool {
	_, ok := typ.(*parser.StringType)
	return ok
}

func isStringish(typ parser.DataType) bool {
	return isCharArray(typ) || isString(typ)
}

func isInteger(typ parser.DataType) bool {
	_, ok := typ.(*parser.IntegerType)
	return ok
}

func assignment(stmt *parser.AssignmentStatement) string {
	if isCharArray(stmt.LeftExpr.Type()) {
		if isCharArray(stmt.RightExpr.Type()) {
			return fmt.Sprintf("copy(%s[:], %s[:])", toExpr(stmt.LeftExpr), toExpr(stmt.RightExpr))
		}
		if isString(stmt.RightExpr.Type()) {
			return fmt.Sprintf("copy(%s[:], []byte(%s))", toExpr(stmt.LeftExpr), toExpr(stmt.RightExpr))
		}
	} else if isString(stmt.LeftExpr.Type()) {
		if isCharArray(stmt.RightExpr.Type()) {
			return fmt.Sprintf("%s = string(%s[:])", toExpr(stmt.LeftExpr), toExpr(stmt.RightExpr))
		}
	}

	return fmt.Sprintf("%s = %s", toExpr(stmt.LeftExpr), toExpr(stmt.RightExpr))
}

func isBooleanType(dt parser.DataType) bool {
	return parser.IsBooleanType(dt)
}

func booleanForLoop(stmt *parser.ForStatement) string {
	rangeFunc := "BoolRange"
	if stmt.DownTo {
		rangeFunc = "BoolRangeDown"
	}
	return fmt.Sprintf("for _, %s := range system.%s(%s, %s)", stmt.Name, rangeFunc, toExpr(stmt.InitialExpr), toExpr(stmt.FinalExpr))
}
