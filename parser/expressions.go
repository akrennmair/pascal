package parser

type expression interface {
	// TODO
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

type relationalExpr struct {
	left     *simpleExpression
	operator itemType
	right    *simpleExpression
}

type minusExpr struct {
	expr *termExpr
}

func isAdditionOperator(typ itemType) bool {
	return typ == itemPlus ||
		typ == itemMinus ||
		typ == itemOr
}

type simpleExpression struct {
	sign  *itemType
	first *termExpr
	next  []*addition
}

type addition struct {
	operator itemType
	term     *termExpr
}

type termExpr struct {
	first factorExpr
	next  []*multiplication
	// TODO: implement
}

type multiplication struct {
	operator itemType
	factor   factorExpr
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

type identifierExpr struct {
	name string
}

type integerExpr struct {
	val int64
}

type floatExpr struct {
	minus       bool
	beforeComma string
	afterComma  string
	scaleFactor int
}

type stringExpr struct {
	str string
}

type nilExpr struct{}

type notExpr struct {
	expr factorExpr
}

type setExpr struct {
	elements []expression
}

type subExpr struct {
	expr expression
}
