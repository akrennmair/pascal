package parser

import (
	"fmt"
	"strings"
)

type expression interface {
	String() string
	Type() dataType
	IsVariableExpr() bool
	Reduce() expression // reduce nested expressions to innermost single expression
}

func isRelationalOperator(typ itemType) bool {
	return typ == itemEqual ||
		typ == itemNotEqual ||
		typ == itemLess ||
		typ == itemLessEqual ||
		typ == itemGreater ||
		typ == itemGreaterEqual ||
		typ == itemIn
}

type relationalOperator string

const (
	opEqual        relationalOperator = "="
	opNotEqual     relationalOperator = "<>"
	opLess         relationalOperator = "<"
	opLessEqual    relationalOperator = "<="
	opGreater      relationalOperator = ">"
	opGreaterEqual relationalOperator = ">="
	opIn           relationalOperator = "in"
)

var itemTypeRelOps = map[itemType]relationalOperator{
	itemEqual:        opEqual,
	itemNotEqual:     opNotEqual,
	itemLess:         opLess,
	itemLessEqual:    opLessEqual,
	itemGreater:      opGreater,
	itemGreaterEqual: opGreaterEqual,
	itemIn:           opIn,
}

func itemTypeToRelationalOperator(typ itemType) relationalOperator {
	op, ok := itemTypeRelOps[typ]
	if !ok {
		return relationalOperator(fmt.Sprintf("INVALID(%d)", typ))
	}
	return op
}

type relationalExpr struct {
	left     expression
	operator relationalOperator
	right    expression
}

func (e *relationalExpr) String() string {
	return fmt.Sprintf("relation<%s %s %s>", e.left, e.operator, e.right)
}

func (e *relationalExpr) Type() dataType {
	return &booleanType{}
}

func (e *relationalExpr) IsVariableExpr() bool {
	return false
}

func (e *relationalExpr) Reduce() expression {
	return &relationalExpr{
		left:     e.left.Reduce(),
		operator: e.operator,
		right:    e.right.Reduce(),
	}
}

type minusExpr struct {
	expr *termExpr
}

func (e *minusExpr) String() string {
	return fmt.Sprintf("minus<%s>", e.expr)
}

func (e *minusExpr) Type() dataType {
	return e.expr.Type()
}

func (e *minusExpr) IsVariableExpr() bool {
	return false
}

func (e *minusExpr) Reduce() expression {
	return e
}

func isAdditionOperator(typ itemType) bool {
	return typ == itemSign || typ == itemOr
}

type simpleExpression struct {
	sign  string
	first expression
	next  []*addition
}

func (e *simpleExpression) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "simple-expr:<%s %s", e.sign, e.first)
	for _, add := range e.next {
		fmt.Fprintf(&buf, " %s %s", add.operator, add.term)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *simpleExpression) Type() dataType {
	return e.first.Type()
}

func (e *simpleExpression) IsVariableExpr() bool {
	if e.sign != "" || len(e.next) > 0 {
		return false
	}

	return e.first.IsVariableExpr()
}

func (e *simpleExpression) Reduce() expression {
	if len(e.next) == 0 && (e.sign == "" || e.sign == "+") {
		return e.first.Reduce()
	}
	ne := &simpleExpression{
		sign:  e.sign,
		first: e.first.Reduce(),
	}

	for _, add := range e.next {
		ne.next = append(ne.next, &addition{
			operator: add.operator,
			term:     add.term.Reduce(),
		})
	}

	return ne
}

type additionOperator string

const (
	opAdd      additionOperator = "+"
	opSubtract additionOperator = "-"
	opOr       additionOperator = "or"
)

func tokenToAdditionOperator(t item) additionOperator {
	if t.typ == itemSign && (t.val == "+" || t.val == "-") {
		return additionOperator(t.val)
	}
	if t.typ == itemOr {
		return opOr
	}
	return additionOperator(fmt.Sprintf("INVALID(%+v)", t))
}

type addition struct {
	operator additionOperator
	term     expression
}

type termExpr struct {
	first factorExpr
	next  []*multiplication
}

func (e *termExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "term-expr:<%s", e.first)
	for _, mul := range e.next {
		fmt.Fprintf(&buf, " %s %s", mul.operator, mul.factor)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *termExpr) Type() dataType {
	return e.first.Type()
}

func (e *termExpr) IsVariableExpr() bool {
	if len(e.next) > 0 {
		return false
	}

	return e.first.IsVariableExpr()
}

func (e *termExpr) Reduce() expression {
	if len(e.next) == 0 {
		return e.first.Reduce()
	}

	ne := &termExpr{
		first: e.first.Reduce(),
	}

	for _, mul := range ne.next {
		ne.next = append(ne.next, &multiplication{
			operator: mul.operator,
			factor:   mul.factor.Reduce(),
		})
	}
	return e
}

type multiplication struct {
	operator multiplicationOperator
	factor   factorExpr
}

type multiplicationOperator string

const (
	opMultiply    multiplicationOperator = "*"
	opFloatDivide multiplicationOperator = "/"
	opDivide      multiplicationOperator = "div"
	opModulo      multiplicationOperator = "mod"
	opAnd         multiplicationOperator = "and"
)

var itemTypMulOp = map[itemType]multiplicationOperator{
	itemMultiply:    opMultiply,
	itemFloatDivide: opFloatDivide,
	itemDiv:         opDivide,
	itemMod:         opModulo,
	itemAnd:         opAnd,
}

func itemTypeToMultiplicationOperator(typ itemType) multiplicationOperator {
	op, ok := itemTypMulOp[typ]
	if !ok {
		return multiplicationOperator(fmt.Sprintf("INVALID(%+v)", typ))
	}
	return op
}

func isMultiplicationOperator(typ itemType) bool {
	return typ == itemMultiply ||
		typ == itemFloatDivide ||
		typ == itemDiv ||
		typ == itemMod ||
		typ == itemAnd
}

type factorExpr interface {
	expression
}

type constantExpr struct {
	name string
	typ  dataType
}

func (e *constantExpr) String() string {
	return fmt.Sprintf("constant:<%s : %s>", e.name, e.typ.Type())
}

func (e *constantExpr) Type() dataType {
	return e.typ
}

func (e *constantExpr) IsVariableExpr() bool {
	return false
}

func (e *constantExpr) Reduce() expression {
	return e
}

type variableExpr struct {
	name string
	typ  dataType
}

func (e *variableExpr) String() string {
	return fmt.Sprintf("variable:<%s>", e.name)
}

func (e *variableExpr) Type() dataType {
	return e.typ
}

func (e *variableExpr) IsVariableExpr() bool {
	return true
}

func (e *variableExpr) Reduce() expression {
	return e
}

type integerExpr struct {
	val int64
}

func (e *integerExpr) String() string {
	return fmt.Sprintf("int:<%d>", e.val)
}

func (e *integerExpr) Type() dataType {
	return &integerType{}
}

func (e *integerExpr) IsVariableExpr() bool {
	return false
}

func (e *integerExpr) Reduce() expression {
	return e
}

type floatExpr struct {
	minus       bool
	beforeComma string
	afterComma  string
	scaleFactor int
}

func (e *floatExpr) String() string {
	sign := ""
	if e.minus {
		sign = "-"
	}
	return fmt.Sprintf("float:<%s%s.%se%d>", sign, e.beforeComma, e.afterComma, e.scaleFactor)
}

func (e *floatExpr) Type() dataType {
	return &realType{}
}

func (e *floatExpr) IsVariableExpr() bool {
	return false
}

func (e *floatExpr) Reduce() expression {
	return e
}

type stringExpr struct {
	str string
}

func (e *stringExpr) String() string {
	return fmt.Sprintf("str:<%q>", e.str)
}

func (e *stringExpr) Type() dataType {
	return getBuiltinType("string")
}

func (e *stringExpr) IsVariableExpr() bool {
	return false
}

func (e *stringExpr) Reduce() expression {
	return e
}

func (e *stringExpr) IsCharLiteral() bool {
	// TODO: solve this neater.
	return len(e.str) == 1 ||
		(len(e.str) == 3 && e.str[0] == '\'' && e.str[2] == '\'') ||
		e.str == "''''"
}

type nilExpr struct{}

func (e *nilExpr) String() string {
	return "nil"
}

func (e *nilExpr) Type() dataType {
	return &pointerType{typ: nil} // nil means it's compatible with any type
}

func (e *nilExpr) IsVariableExpr() bool {
	return false
}

func (e *nilExpr) Reduce() expression {
	return e
}

type notExpr struct {
	expr factorExpr
}

func (e *notExpr) String() string {
	return fmt.Sprintf("not:<%s>", e.expr)
}

func (e *notExpr) Type() dataType {
	return e.expr.Type()
}

func (e *notExpr) IsVariableExpr() bool {
	return false
}

func (e *notExpr) Reduce() expression {
	return e
}

type setExpr struct {
	elements []expression
}

func (e *setExpr) String() string {
	var buf strings.Builder
	fmt.Fprint(&buf, "set-expr:<")
	for idx, elem := range e.elements {
		if idx > 0 {
			fmt.Fprint(&buf, " ")
		}
		fmt.Fprintf(&buf, "%s", elem)
	}
	fmt.Fprint(&buf, ">")
	return buf.String()
}

func (e *setExpr) Type() dataType {
	return &setType{
		elementType: e.elements[0].Type(),
	}
}

func (e *setExpr) IsVariableExpr() bool {
	return false
}

func (e *setExpr) Reduce() expression {
	return e
}

type subExpr struct {
	expr expression
}

func (e *subExpr) String() string {
	return fmt.Sprintf("sub-expr:<%s>", e.expr)
}

func (e *subExpr) Type() dataType {
	return e.expr.Type()
}

func (e *subExpr) IsVariableExpr() bool {
	return e.expr.IsVariableExpr() // TODO: check whether this is correct.
}

func (e *subExpr) Reduce() expression {
	return e.expr.Reduce()
}

type indexedVariableExpr struct {
	expr  expression // and expression of type *arrayType
	typ   dataType
	exprs []expression
}

func (e *indexedVariableExpr) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "indexed-variable-expr:<(%s)[", e.expr.String())
	for idx, expr := range e.exprs {
		if idx > 0 {
			fmt.Fprint(&buf, ", ")
		}
		fmt.Fprintf(&buf, "%s", expr)
	}
	fmt.Fprint(&buf, "]>")
	return buf.String()
}

func (e *indexedVariableExpr) Type() dataType {
	return e.typ
}

func (e *indexedVariableExpr) IsVariableExpr() bool {
	return true
}

func (e *indexedVariableExpr) Reduce() expression {
	ne := &indexedVariableExpr{
		expr: e.expr.Reduce(),
		typ:  e.typ,
	}
	for _, ie := range e.exprs {
		ne.exprs = append(ne.exprs, ie.Reduce())
	}
	return ne
}

type functionCallExpr struct {
	name   string
	typ    dataType
	params []expression
}

func (e *functionCallExpr) String() string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "function-call-expr:<%s(", e.name)
	for idx, param := range e.params {
		if idx > 0 {
			fmt.Fprintf(&buf, ", ")
		}
		fmt.Fprint(&buf, param)
	}
	fmt.Fprint(&buf, ")>")
	return buf.String()
}

func (e *functionCallExpr) Type() dataType {
	return e.typ
}

func (e *functionCallExpr) IsVariableExpr() bool {
	return false
}

func (e *functionCallExpr) Reduce() expression {
	ne := &functionCallExpr{
		name: e.name,
		typ:  e.typ,
	}

	for _, pe := range ne.params {
		ne.params = append(ne.params, pe.Reduce())
	}

	return ne
}

type fieldDesignatorExpr struct {
	expr  expression
	field string
	typ   dataType
}

func (e *fieldDesignatorExpr) String() string {
	return fmt.Sprintf("field-designator-expr:<%s.%s>", e.expr, e.field)
}

func (e *fieldDesignatorExpr) Type() dataType {
	return e.typ
}

func (e *fieldDesignatorExpr) IsVariableExpr() bool {
	return true
}

func (e *fieldDesignatorExpr) Reduce() expression {
	return &fieldDesignatorExpr{
		expr:  e.expr.Reduce(),
		field: e.field,
		typ:   e.typ,
	}
}

type enumValueExpr struct {
	symbol string
	value  int
	typ    dataType
}

func (e *enumValueExpr) String() string {
	return fmt.Sprintf("enum-value-expr:<%s %d of type %s>", e.symbol, e.value, e.typ.Type())
}

func (e *enumValueExpr) Type() dataType {
	return e.typ
}

func (e *enumValueExpr) IsVariableExpr() bool {
	return true
}

func (e *enumValueExpr) Reduce() expression {
	return e
}

type derefExpr struct {
	expr expression
}

func (e *derefExpr) String() string {
	return fmt.Sprintf("deref-expr:<%s>", e.expr)
}

func (e *derefExpr) Type() dataType {
	t, ok := e.expr.Type().(*pointerType)
	if !ok {
		panic("derefExpr was created with expression not of pointer type")
	}
	return t.typ
}

func (e *derefExpr) IsVariableExpr() bool {
	return e.expr.IsVariableExpr()
}

func (e *derefExpr) Reduce() expression {
	return &derefExpr{
		expr: e.expr.Reduce(),
	}
}

type formatExpr struct {
	expr          expression
	width         expression // TODO: find out what param1 and param2 stand for and rename them
	decimalPlaces expression
}

func (e *formatExpr) String() string {
	var buf strings.Builder

	buf.WriteString("format-expr:<")
	buf.WriteString(e.expr.String())
	if e.width != nil {
		buf.WriteString(":")
		buf.WriteString(e.width.String())
	}
	if e.decimalPlaces != nil {
		buf.WriteString(":")
		buf.WriteString(e.decimalPlaces.String())
	}
	buf.WriteString(">")
	return buf.String()
}

func (e *formatExpr) Type() dataType {
	return e.expr.Type()
}

func (e *formatExpr) IsVariableExpr() bool {
	return e.expr.IsVariableExpr()
}

func (e *formatExpr) Reduce() expression {
	return e
}
