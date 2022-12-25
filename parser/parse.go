package parser

import (
	"errors"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
	"strings"
)

func NewParser(name, text string) *program {
	return &program{
		lexer:  lex(name, text),
		logger: log.New(io.Discard, "parser", log.LstdFlags|log.Lshortfile),
	}
}

func (p *program) SetLogOutput(w io.Writer) {
	p.logger.SetOutput(w)
}

func (p *program) Parse() (err error) {
	defer p.recover(&err)
	err = p.parse()
	return err
}

func parse(name, text string) (p *program, err error) {
	return parseWithLexer(lex(name, text))
}

func parseWithLexer(lexer *lexer) (p *program, err error) {
	p = &program{
		lexer:  lexer,
		logger: log.New(io.Discard, "parser", log.LstdFlags|log.Lshortfile),
	}
	defer p.recover(&err)
	err = p.parse()
	return p, err
}

type program struct {
	lexer     *lexer
	logger    *log.Logger
	token     [3]item
	peekCount int

	name  string
	block *block
}

func (p *program) recover(errp *error) {
	e := recover()
	if e != nil {
		// rethrow runtime errors
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
}

func (p *program) backup() {
	p.peekCount++
}

func (p *program) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lexer.nextItem()
	return p.token[0]
}

func (p *program) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lexer.nextItem()
	}
	return p.token[p.peekCount]
}

func (p *program) errorf(fmtstr string, args ...interface{}) {
	err := errors.New(fmt.Sprintf("%s:%d: ", p.lexer.name, p.lexer.lineNumber()) + fmt.Sprintf(fmtstr, args...))
	panic(err)
}

func (p *program) parse() (err error) {
	defer p.recover(&err)
	p.parseProgramHeading()
	p.block = p.parseBlock(nil, nil)

	if p.peek().typ != itemDot {
		p.errorf("expected ., got %s instead", p.next())
	}
	p.next()

	return nil
}

func (p *program) parseProgramHeading() {
	if p.peek().typ != itemProgram {
		p.errorf("expected program, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}
	p.name = p.next().val

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()
}

type block struct {
	parent               *block     // the parent block this block belongs to.
	procedure            *procedure // the procedure or function this block belongs to. nil if topmost program.
	labels               []string
	constantDeclarations []*constDeclaration
	typeDefinitions      []*typeDefinition
	variables            []*variable
	procedures           []*procedure
	functions            []*procedure
	statements           []statement
}

func (b *block) findConstantDeclaration(name string) *constDeclaration {
	if b == nil {
		return nil
	}

	for _, constant := range b.constantDeclarations {
		if constant.Name == name {
			return constant
		}
	}

	return b.parent.findConstantDeclaration(name)
}

func (b *block) findVariable(name string) *variable {
	if b == nil {
		return nil
	}

	for _, variable := range b.variables {
		if variable.Name == name {
			return variable
		}
	}

	return b.parent.findVariable(name)
}

func (b *block) findFormalParameter(name string) *formalParameter {
	if b == nil {
		return nil
	}

	if b.procedure == nil {
		return b.parent.findFormalParameter(name)
	}

	for _, param := range b.procedure.FormalParameters {
		if param.Name == name {
			return param
		}
	}

	return b.parent.findFormalParameter(name)
}

func (b *block) findProcedure(name string) *procedure {
	if b == nil {
		return findBuiltinProcedure(name)
	}

	for _, proc := range b.procedures {
		if proc.Name == name {
			return proc
		}
	}

	return b.parent.findProcedure(name)
}

func (b *block) findFunction(name string) *procedure {
	if b == nil {
		return nil
	}

	for _, proc := range b.functions {
		if proc.Name == name {
			return proc
		}
	}

	return b.parent.findFunction(name)
}

func (b *block) findFunctionForAssignment(name string) *procedure {
	if b == nil || b.procedure == nil {
		return nil
	}

	if b.procedure.Name == name {
		return b.procedure
	}

	return b.parent.findFunctionForAssignment(name)
}

func (b *block) findType(name string) dataType {
	typ := getBuiltinType(name)
	if typ != nil {
		return typ
	}

	if b == nil {
		return nil
	}

	var foundType dataType

	for _, typ := range b.typeDefinitions {
		if typ.Name == name {
			foundType = typ.Type
			break
		}
	}

	if foundType == nil {
		foundType = b.parent.findType(name)
	}

	return foundType
}

func (b *block) isValidLabel(label string) bool {
	for _, l := range b.labels {
		if l == label {
			return true
		}
	}
	return false
}

func (p *program) parseBlock(parent *block, proc *procedure) *block {
	b := &block{
		parent:    parent,
		procedure: proc,
	}
	p.parseDeclarationPart(b)
	p.parseStatementPart(b)
	return b
}

func (p *program) parseDeclarationPart(b *block) {
	if p.peek().typ == itemLabel {
		p.parseLabelDeclarationPart(b)
	}
	if p.peek().typ == itemConst {
		p.parseConstDeclarationPart(b)
	}
	if p.peek().typ == itemTyp {
		p.parseTypeDeclarationPart(b)
	}
	if p.peek().typ == itemVar {
		p.parseVarDeclarationPart(b)
	}
	p.parseProcedureAndFunctionDeclarationPart(b)
}

func (p *program) parseStatementPart(b *block) {
	if p.peek().typ != itemBegin {
		p.errorf("expected begin, got %s instead", p.next())
	}
	p.next()

	b.statements = p.parseStatementSequence(b)

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead", p.next())
	}
	p.next()
}

func (p *program) parseLabelDeclarationPart(b *block) {
	if p.peek().typ != itemLabel {
		p.errorf("expected label, got %s", p.next())
	}
	p.next()

	b.labels = []string{}

labelDeclarationLoop:
	for {

		if p.peek().typ != itemUnsignedDigitSequence {
			p.errorf("expected number, got %s", p.next())
		}
		b.labels = append(b.labels, p.next().val)

		switch p.peek().typ {
		case itemComma:
			p.next()
		case itemSemicolon:
			p.next()
			break labelDeclarationLoop
		default:
			p.errorf("expected comma or semicolon, got %s", p.next())
		}
	}
}

func (p *program) parseConstDeclarationPart(b *block) {
	if p.peek().typ != itemConst {
		p.errorf("expected const, got %s", p.next())
	}
	p.next()

	b.constantDeclarations = []*constDeclaration{}

	constDecl, ok := p.parseConstantDefinition(b)
	if !ok {
		p.errorf("expected constant definition")
	}
	b.constantDeclarations = append(b.constantDeclarations, constDecl)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for {
		constDecl, ok := p.parseConstantDefinition(b)
		if !ok {
			break
		}

		b.constantDeclarations = append(b.constantDeclarations, constDecl)
		if p.peek().typ != itemSemicolon {
			p.errorf("expected semicolon, got %s", p.next())
		}
		p.next()
	}
}

type constDeclaration struct {
	Name  string
	Value int // TODO: support all types of constants
}

func (p *program) parseConstantDefinition(b *block) (*constDeclaration, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	constName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	constValue := p.parseConstant(b)

	return &constDeclaration{Name: constName, Value: constValue}, true
}

func (p *program) parseTypeDeclarationPart(b *block) {
	if p.peek().typ != itemTyp {
		p.errorf("expected type, got %s", p.next())
	}
	p.next()

	b.typeDefinitions = []*typeDefinition{}
	typeDecl, ok := p.parseTypeDefinition(b)
	if !ok {
		p.errorf("expected type definition")
	}
	b.typeDefinitions = append(b.typeDefinitions, typeDecl)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for {
		typeDecl, ok := p.parseTypeDefinition(b)
		if !ok {
			break
		}

		b.typeDefinitions = append(b.typeDefinitions, typeDecl)
		if p.peek().typ != itemSemicolon {
			p.errorf("expected semicolon, got %s", p.next())
		}
		p.next()
	}
}

type typeDefinition struct {
	Name string
	Type dataType
}

func (p *program) parseTypeDefinition(b *block) (*typeDefinition, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	typeName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	dataType := p.parseDataType(b)

	return &typeDefinition{Name: typeName, Type: dataType}, true
}

type dataType interface {
	Type() string
	Equals(dt dataType) bool
}

type recordField struct {
	Identifiers []string
	Type        dataType
}

func (f *recordField) String() string {
	var buf strings.Builder
	for idx, id := range f.Identifiers {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(id)
	}
	buf.WriteString(" : ")
	buf.WriteString(f.Type.Type())
	return buf.String()
}

func (p *program) parseDataType(b *block) dataType {
	packed := false

restartParseDataType:
	switch p.peek().typ {
	case itemIdentifier:
		ident := p.next().val
		if typ := b.findType(ident); typ != nil {
			return typ
		}
		p.errorf("unknown type %s", ident)
	case itemCaret:
		p.next() // skip ^ token.
		if p.peek().typ != itemIdentifier {
			p.errorf("expected type after ^, got %s", p.next())
		}
		return &pointerType{name: p.next().val}
	case itemOpenParen:
		return p.parseEnumType(b)
	case itemPacked: // TODO: ensure that packed only appears before structured types (array, record, set, file)
		if packed {
			p.errorf("expected type after packed, got %s", p.next())
		}
		p.next()
		packed = true
		goto restartParseDataType
	case itemArray:
		return p.parseArrayType(b, packed)
	case itemRecord:
		return p.parseRecordType(b, packed)
	case itemSet:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after set, got %s", p.next())
		}
		p.next()
		setDataType := p.parseDataType(b)
		return &setType{elementType: setDataType, packed: packed}
	case itemFile:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after file, got %s", p.next())
		}
		p.next()
		fileDataType := p.parseDataType(b)
		_ = fileDataType
		return nil // TODO: implement file type, including packed
	default:
		p.errorf("unknown type %s", p.next().val)
	}
	// not reached.
	return nil
}

func (p *program) parseEnumType(b *block) *enumType {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	identifierList := p.parseIdentifierList(b)

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s", p.next())
	}
	p.next()

	return &enumType{identifiers: identifierList}
}

func (p *program) parseIdentifierList(b *block) []string {
	identifierList := []string{}

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}
	identifierList = append(identifierList, p.next().val)

	for {
		if p.peek().typ != itemComma {
			break
		}
		p.next()

		if p.peek().typ != itemIdentifier {
			p.errorf("expected identifier, got %s", p.next())
		}
		identifierList = append(identifierList, p.next().val)
	}

	return identifierList
}

type variable struct {
	Name string
	Type dataType
}

func (p *program) parseVarDeclarationPart(b *block) {
	if p.peek().typ != itemVar {
		p.errorf("expected var, got %s", p.next())
	}
	p.next()

	for {

		variableNames := p.parseIdentifierList(b)

		if p.peek().typ != itemColon {
			p.errorf("expected :, got %s", p.next())
		}
		p.next()

		dataType := p.parseDataType(b)

		if p.peek().typ != itemSemicolon {
			p.errorf("expected ;, got %s", p.next())
		}
		p.next()

		for _, varName := range variableNames {
			b.variables = append(b.variables, &variable{Name: varName, Type: dataType})
		}

		if p.peek().typ != itemIdentifier {
			break
		}
	}
}

func (p *program) parseProcedureAndFunctionDeclarationPart(b *block) {
	for {
		switch p.peek().typ {
		case itemProcedure:
			p.parseProcedureDeclaration(b)
			if p.peek().typ != itemSemicolon {
				p.errorf("expected ;, got %s", p.next())
			}
			p.next()
		case itemFunction:
			p.parseFunctionDeclaration(b)
			if p.peek().typ != itemSemicolon {
				p.errorf("expected ;, got %s", p.next())
			}
			p.next()
		default:
			return
		}
	}
}

type procedure struct {
	Name             string
	Block            *block
	FormalParameters []*formalParameter
	ReturnType       dataType
	varargs          bool // for builtin functions with variable arguments.
}

func (p *program) parseProcedureDeclaration(b *block) {
	if p.peek().typ != itemProcedure {
		p.errorf("expected procedure, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var parameterList []*formalParameter
	if p.peek().typ == itemOpenParen {
		parameterList = p.parseFormalParameterList(b)
	}

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &procedure{Name: procedureName, FormalParameters: parameterList}

	proc.Block = p.parseBlock(b, proc)

	b.procedures = append(b.procedures, proc)
}

type formalParameter struct {
	Name           string
	Type           dataType
	ValueParameter bool
}

func (p *program) parseFormalParameterList(b *block) []*formalParameter {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	parameterList := []*formalParameter{}

parameterListLoop:
	for {

		formalParameters := p.parseFormalParameter(b)

		parameterList = append(parameterList, formalParameters...)

		switch p.peek().typ {
		case itemSemicolon:
			p.next()
			continue parameterListLoop
		case itemCloseParen:
			p.next()
			break parameterListLoop
		default:
			p.errorf("expected ; or ), got %s", p.next())
		}
	}

	return parameterList
}

func (p *program) parseFormalParameter(b *block) []*formalParameter {
	valueParameter := false
	if p.peek().typ == itemVar {
		valueParameter = true
		p.next()
	}

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	parameterNames := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	parameterType := p.parseDataType(b)

	formalParameters := make([]*formalParameter, 0, len(parameterNames))
	for _, name := range parameterNames {
		formalParameters = append(formalParameters, &formalParameter{Name: name, Type: parameterType, ValueParameter: valueParameter})
	}

	return formalParameters
}

func (p *program) parseFunctionDeclaration(b *block) {
	if p.peek().typ != itemFunction {
		p.errorf("expected function, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var parameterList []*formalParameter
	if p.peek().typ == itemOpenParen {
		parameterList = p.parseFormalParameterList(b)
	}

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	returnType := p.parseDataType(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &procedure{Name: procedureName, FormalParameters: parameterList, ReturnType: returnType}

	proc.Block = p.parseBlock(b, proc)

	b.functions = append(b.functions, proc)
}

func (p *program) parseStatementSequence(b *block) []statement {
	var statements []statement

	first := true

	for {
		if p.peek().typ == itemEnd || p.peek().typ == itemUntil {
			break
		}

		if !first {
			if p.peek().typ != itemSemicolon {
				break
			}
			p.next()
		}

		stmt := p.parseStatement(b)

		statements = append(statements, stmt)

		first = false
	}

	return statements
}

type statementType int

const (
	stmtGoto statementType = iota
	stmtAssignment
	stmtProcedureCall
	stmtCompoundStatement
	stmtWhile
	stmtRepeat
	stmtFor
	stmtIf
)

type statement interface {
	Type() statementType
	Label() *string
}

func (p *program) parseStatement(b *block) statement {
	var label *string
	if p.peek().typ == itemUnsignedDigitSequence {
		l := p.next().val

		if !b.isValidLabel(l) {
			p.errorf("invalid label %s", l)
		}
		label = &l

		if p.peek().typ != itemColon {
			p.errorf("expected : after label, got %s", p.next())
		}
		p.next()
	}

	switch p.peek().typ {
	case itemGoto:
		p.next()
		if p.peek().typ != itemUnsignedDigitSequence {
			p.errorf("expected label after goto, got %s", p.next())
		}
		tl := p.next().val
		if !b.isValidLabel(tl) {
			p.errorf("invalid goto label %s", tl)
		}
		return &statementGoto{label: label, target: tl}
	case itemIdentifier:
		return p.parseAssignmentOrProcedureStatement(b)
	case itemBegin:
		p.next()
		statements := p.parseStatementSequence(b)
		if p.peek().typ != itemEnd {
			p.errorf("expected end, got %s", p.next())
		}
		p.next()
		return &compoundStatement{statements: statements}
	case itemWhile:
		return p.parseWhileStatement(b)
	case itemRepeat:
		return p.parseRepeatStatement(b)
	case itemFor:
		return p.parseForStatement(b)
	case itemIf:
		return p.parseIfStatement(b)
	case itemCase:
		return p.parseCaseStatement(b)
		// TODO: implement with statement.
	}
	p.errorf("unsupported %s as statement", p.next())
	return nil
}

func (p *program) parseAssignmentOrProcedureStatement(b *block) statement {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	identifier := p.next().val

	var lexpr expression

	switch p.peek().typ {
	case itemOpenBracket:
		p.next()
		indexes := p.parseExpressionList(b) // TODO: validate that all expressions are of the right type for indexing an array.
		if p.peek().typ != itemCloseBracket {
			p.errorf("expected ], got %s instead", p.peek())
		}
		p.next()
		varDecl := b.findVariable(identifier)
		if varDecl == nil {
			p.errorf("unknown variable %s", identifier)
		}
		at, ok := varDecl.Type.(*arrayType) // TODO: support string
		if !ok {
			p.errorf("variable %s is not an array", identifier)
		}
		// TODO: check whether variable is an array, and whether dimensions fit.
		lexpr = &indexedVariableExpr{name: identifier, exprs: indexes, typ: at.elementType}
	case itemDot:
		varDecl := b.findVariable(identifier)
		if varDecl == nil {
			p.errorf("unknown variable %s", identifier)
		}
		p.next()
		if p.peek().typ != itemIdentifier {
			p.errorf("expected field identifier, got %s instead", p.peek())
		}
		fieldIdentifier := p.next().val
		rt, ok := varDecl.Type.(*recordType)
		if !ok {
			p.errorf("variable %s is not a record", identifier)
		}
		field := rt.findField(fieldIdentifier)
		if field == nil {
			p.errorf("field %s.%s does not exist", identifier, fieldIdentifier)
		}
		lexpr = &fieldDesignatorExpr{name: identifier, field: fieldIdentifier, typ: field.Type}
	case itemOpenParen:
		proc := b.findProcedure(identifier)
		if proc == nil {
			p.errorf("unknown procedure %s", identifier)
		}
		actualParameterList := p.parseActualParameterList(b)
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, actualParameterList); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &procedureCallStatement{name: identifier, parameterList: actualParameterList}
	default:
		if varDecl := b.findVariable(identifier); varDecl != nil {
			lexpr = &variableExpr{name: identifier, typ: varDecl.Type}
		} else if funcDecl := b.findFunctionForAssignment(identifier); funcDecl != nil {
			lexpr = &variableExpr{name: identifier, typ: funcDecl.ReturnType} // TODO: do we need a separate expression type for this?
		}
	}

	switch p.peek().typ {
	case itemAssignment:
		p.next()
		if lexpr == nil {
			p.errorf("assignment: unknown left expression %s", identifier)
		}
		rexpr := p.parseExpression(b)
		return &assignmentStatement{lexpr: lexpr, rexpr: rexpr}
	default:
		// TODO: improve handling if we hit this branch, but previously had an indexed-variable or field-designator.
		proc := b.findProcedure(identifier)
		if proc == nil {
			p.errorf("unknown procedure %s", identifier)
		}
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, []expression{}); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &procedureCallStatement{name: identifier}
	}
}

func (p *program) parseWhileStatement(b *block) *whileStatement {
	if p.peek().typ != itemWhile {
		p.errorf("expected while, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)

	if !condition.Type().Equals(&booleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &whileStatement{condition: condition, stmt: stmt}
}

func (p *program) parseRepeatStatement(b *block) *repeatStatement {
	if p.peek().typ != itemRepeat {
		p.errorf("expected repeat, got %s", p.next())
	}
	p.next()

	stmts := p.parseStatementSequence(b)

	if p.peek().typ != itemUntil {
		p.errorf("expected until, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)
	if !condition.Type().Equals(&booleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	return &repeatStatement{condition: condition, stmts: stmts}
}

func (p *program) parseForStatement(b *block) *forStatement {
	if p.peek().typ != itemFor {
		p.errorf("expected for, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected variable-identifier, got %s", p.next())
	}

	variable := p.next().val

	if b.findVariable(variable) == nil {
		p.errorf("unknown variable %s", variable)
	}

	if p.peek().typ != itemAssignment {
		p.errorf("expected :=, got %s", p.next())
	}
	p.next()

	initialExpr := p.parseExpression(b)

	down := false
	switch p.peek().typ {
	case itemTo:
		down = false
	case itemDownto:
		down = true
	default:
		p.errorf("expected to or downto, got %s", p.next())
	}
	p.next()

	finalExpr := p.parseExpression(b)

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &forStatement{name: variable, initialExpr: initialExpr, finalExpr: finalExpr, body: stmt, down: down}
}

func (p *program) parseIfStatement(b *block) *ifStatement {
	if p.peek().typ != itemIf {
		p.errorf("expected if, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)
	if !condition.Type().Equals(&booleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	if p.peek().typ != itemThen {
		p.errorf("expected then, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	var elseStmt statement

	if p.peek().typ == itemElse {
		p.next()
		elseStmt = p.parseStatement(b)
	}

	return &ifStatement{condition: condition, body: stmt, elseBody: elseStmt}
}

func (p *program) parseCaseStatement(b *block) statement {
	p.errorf("TODO: case not implemented")
	// TODO: implement.
	return nil
}

func (p *program) parseActualParameterList(b *block) []expression {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	params := []expression{}

	for {
		expr := p.parseExpression(b)

		params = append(params, expr)

		if p.peek().typ != itemComma {
			break
		}
		p.next()
	}

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s", p.next())
	}
	p.next()

	return params
}

type exprType int

const (
	exprIdentifier exprType = iota
	exprNumber
)

func (p *program) parseExpression(b *block) expression {
	p.logger.Printf("Parsing expression")

	expr := p.parseSimpleExpression(b)
	if !isRelationalOperator(p.peek().typ) {
		return expr
	}

	operator := itemTypeToRelationalOperator(p.next().typ)

	p.logger.Printf("Found relational operator %s after first simple expression", operator)

	rightExpr := p.parseSimpleExpression(b)

	p.logger.Printf("Finished parsing expression")

	relExpr := &relationalExpr{
		left:     expr,
		operator: operator,
		right:    rightExpr,
	}

	lt := relExpr.left.Type()
	rt := relExpr.right.Type()
	if operator == opIn {
		st, ok := rt.(*setType)
		if !ok {
			p.errorf("in: expected set type, got %s instead.", rt)
		}
		if !lt.Equals(st.elementType) {
			p.errorf("type %s does not match set type %s", lt.Type(), st.elementType.Type())
		}
	} else {
		ok := lt.Equals(rt)
		if !ok {
			p.errorf("can't %s %s %s", lt.Type(), relExpr.operator, rt.Type())
		}
	}

	return relExpr
}

func (p *program) parseSimpleExpression(b *block) *simpleExpression {
	p.logger.Printf("Parsing simple expression")
	var sign string
	if typ := p.peek().typ; typ == itemSign {
		sign = p.next().val
	}

	term := p.parseTerm(b)

	simpleExpr := &simpleExpression{
		sign:  sign,
		first: term,
	}

	if !isAdditionOperator(p.peek().typ) {
		p.logger.Printf("Finished parsing simple expression without further operators because %s is not an addition operator", p.peek())
		return simpleExpr
	}

	for {
		operatorToken := p.next()

		operator := tokenToAdditionOperator(operatorToken)

		if operator == opOr {
			_, ok := simpleExpr.first.Type().(*booleanType)
			if !ok {
				p.errorf("can't use or with %s", simpleExpr.first.Type().Type())
			}
		} else {
			// TODO: validate whether type is suitable for addition
		}

		nextTerm := p.parseTerm(b)

		if !simpleExpr.first.Type().Equals(nextTerm.Type()) {
			p.errorf("can't %s %s %s", simpleExpr.first.Type().Type(), operator, nextTerm.Type().Type())
		}

		simpleExpr.next = append(simpleExpr.next, &addition{operator: operator, term: nextTerm})

		if !isAdditionOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing simple expression because %s is not an addition operator", p.peek())

	return simpleExpr
}

func (p *program) parseTerm(b *block) *termExpr {
	p.logger.Printf("Parsing term")
	factor := p.parseFactor(b)

	term := &termExpr{
		first: factor,
	}

	if !isMultiplicationOperator(p.peek().typ) {
		p.logger.Printf("Finished parsing term without further operator because %s is not a multiplication operator", p.peek())
		return term
	}

	for {
		operator := itemTypeToMultiplicationOperator(p.next().typ)
		p.logger.Printf("parseTerm: got operator %s", operator)

		if operator == opAnd {
			_, ok := term.first.Type().(*booleanType)
			if !ok {
				p.errorf("can't use and with %s", term.first.Type().Type())
			}
		} else {
			// TODO: validate whether type is suitable for multiplication
		}

		nextFactor := p.parseFactor(b)

		if !term.first.Type().Equals(nextFactor.Type()) {
			p.errorf("can't %s %s %s", term.first.Type().Type(), operator, nextFactor.Type().Type())
		}

		term.next = append(term.next, &multiplication{operator: operator, factor: nextFactor})

		if !isMultiplicationOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing term because %s is not a multiplication operator", p.peek())

	return term
}

func (p *program) parseFactor(b *block) factorExpr {
	p.logger.Printf("Parsing factor")
	defer p.logger.Printf("Finished parsing factor")

	switch p.peek().typ {
	case itemIdentifier:
		p.logger.Printf("parseFactor: got identifier %s", p.peek().val)
		ident := p.next().val
		switch p.peek().typ {
		case itemOpenBracket:
			varDecl := b.findVariable(ident)
			if varDecl == nil {
				p.errorf("unknown variable %s", ident)
			}
			p.next()
			expressions := p.parseExpressionList(b) // TODO: validate that all expressions are of the right type for indexing an array.
			if p.peek().typ != itemCloseBracket {
				p.errorf("expected ], got %s instead", p.peek())
			}
			p.next()
			at, ok := varDecl.Type.(*arrayType)
			if !ok {
				p.errorf("variable %s is not an array", ident)
			}
			return &indexedVariableExpr{name: ident, exprs: expressions, typ: at.elementType}
		case itemOpenParen:
			funcDecl := b.findFunction(ident)
			if funcDecl == nil {
				p.errorf("unknown function %s", ident)
			}
			params := p.parseActualParameterList(b)
			if err := p.validateParameters(funcDecl.varargs, funcDecl.FormalParameters, params); err != nil {
				p.errorf("function %s: %v", ident, err)
			}
			return &functionCallExpr{name: ident, params: params, typ: funcDecl.ReturnType}
		case itemDot:
			p.next()

			varDecl := b.findVariable(ident)
			if varDecl == nil {
				p.errorf("unknown variable %s", ident)
			}
			rt, ok := varDecl.Type.(*recordType)
			if !ok {
				p.errorf("variable %s is not of a record type", ident)
			}

			if p.peek().typ != itemIdentifier {
				p.errorf("expected identifier, got %s instead", p.peek())
			}
			fieldIdentifier := p.next().val
			field := rt.findField(fieldIdentifier)

			return &fieldDesignatorExpr{name: ident, field: fieldIdentifier, typ: field.Type}
		}
		if funcDecl := b.findFunction(ident); funcDecl != nil {
			if err := p.validateParameters(funcDecl.varargs, funcDecl.FormalParameters, []expression{}); err != nil {
				p.errorf("function %s: %v", ident, err)
			}
			return &functionCallExpr{name: ident, typ: funcDecl.ReturnType}
		}
		if constDecl := b.findConstantDeclaration(ident); constDecl != nil {
			return &constantExpr{ident, &integerType{} /* TODO: constDecl.Type */}
		}
		if varDecl := b.findVariable(ident); varDecl != nil {
			return &variableExpr{name: ident, typ: varDecl.Type}
		}
		if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
			return &variableExpr{name: ident, typ: paramDecl.Type} // TODO: do we need a separate formal parameter expression here?
		}
		p.errorf("unknown identifier %s", ident)
	case itemSign:
		sign := p.next().val
		return p.parseNumber(sign == "-")
	case itemUnsignedDigitSequence:
		return p.parseNumber(false)
	case itemStringLiteral:
		p.logger.Printf("parseFactor: got string literal %s", p.peek())
		return &stringExpr{p.next().val}
	case itemOpenBracket:
		return p.parseSet(b)
	case itemNil:
		p.next()
		return &nilExpr{}
	case itemOpenParen:
		return p.parseSubExpr(b)
	case itemNot:
		p.next()
		expr := p.parseFactor(b)
		if !expr.Type().Equals(&booleanType{}) {
			p.errorf("can't NOT %s", expr.Type().Type())
		}
		return &notExpr{expr}
	default:
		p.errorf("unexpected %s while parsing factor", p.peek())
	}
	// unreachable
	return nil
}

func (p *program) parseNumber(minus bool) factorExpr {
	p.logger.Printf("Parsing number")

	unsignedDigitSequence := p.next().val
	if p.peek().typ == itemDot || (p.peek().typ == itemIdentifier && strings.ToLower(p.peek().val) == "e") {
		scaleFactor := 0
		afterComma := ""
		if p.peek().typ == itemDot {
			p.next()
			if p.peek().typ == itemUnsignedDigitSequence { // N.B. EBNF says digit-sequence here, but this doesn't make sense.
				afterComma = p.next().val
			}
			if p.peek().typ == itemIdentifier && strings.ToLower(p.peek().val) == "e" {
				p.next()
				scaleFactor = p.parseScaleFactor()
			}
		} else if p.peek().typ == itemIdentifier && strings.ToLower(p.peek().val) == "e" { // TODO: change lexing so that e is returned as its own token in this particular instance.
			p.next()
			scaleFactor = p.parseScaleFactor()
		} else {
			p.errorf("expected either . or E, but got %v instead", p.peek())
		}
		p.logger.Printf("parseNumber: parsed float")
		return &floatExpr{minus: minus, beforeComma: unsignedDigitSequence, afterComma: afterComma, scaleFactor: scaleFactor}
	}
	intValue, err := strconv.ParseInt(unsignedDigitSequence, 10, 64)
	if err != nil {
		p.errorf("failed to parse %s as integer: %v", unsignedDigitSequence, err)
	}
	if minus {
		intValue = -intValue
	}
	p.logger.Printf("parseNumber: parsed int %d", intValue)
	return &integerExpr{intValue}
}

func (p *program) parseScaleFactor() int {
	minus := false
	if typ := p.peek().typ; typ == itemSign {
		minus = p.next().val == "-"
	}
	if p.peek().typ != itemUnsignedDigitSequence {
		p.errorf("expected unsigned-digit-sequence, got %v instead", p.peek())
	}
	num := p.next().val
	scaleFactor, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		p.errorf("failed to parse %s as integer: %v", num, err)
	}
	if minus {
		scaleFactor = -scaleFactor
	}
	return int(scaleFactor)
}

func (p *program) parseSet(b *block) *setExpr {
	if p.peek().typ != itemOpenBracket {
		p.errorf("expected [, found %s instead", p.next())
	}
	p.next()

	set := &setExpr{}

	expr := p.parseExpression(b)
	set.elements = append(set.elements, expr)

loop:
	for {
		switch p.peek().typ {
		case itemComma:
			p.next()
		case itemCloseBracket:
			p.next()
			break loop
		default:
			p.errorf("expected , or ], got %s intead", p.peek())
		}

		expr := p.parseExpression(b)
		set.elements = append(set.elements, expr)
	}

	return set
}

func (p *program) parseSubExpr(b *block) *subExpr {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s instead", p.peek())
	}
	p.next()

	expr := p.parseExpression(b)

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s instead", p.peek())
	}
	p.next()

	return &subExpr{expr}
}

func (p *program) parseExpressionList(b *block) []expression {
	var exprs []expression

	expr := p.parseExpression(b)

	exprs = append(exprs, expr)

	for {
		if p.peek().typ != itemComma {
			break
		}
		p.next()

		expr := p.parseExpression(b)

		exprs = append(exprs, expr)
	}

	return exprs
}

func (p *program) parseArrayType(b *block, packed bool) *arrayType {
	if p.peek().typ != itemArray {
		p.errorf("expected array, got %s instead", p.peek())
	}
	p.next()

	if p.peek().typ != itemOpenBracket {
		p.errorf("expected [, got %s instead", p.peek())
	}
	p.next()

	indexTypes := []dataType{p.parseSimpleType(b)}

	for {
		if p.peek().typ != itemComma {
			break
		}
		p.next()

		indexType := p.parseSimpleType(b)

		indexTypes = append(indexTypes, indexType)
	}

	if p.peek().typ != itemCloseBracket {
		p.errorf("expected ], got %s instead", p.peek())
	}
	p.next()

	if p.peek().typ != itemOf {
		p.errorf("expected of, got %s instead", p.peek())
	}
	p.next()

	elementType := p.parseDataType(b)

	return &arrayType{
		indexTypes:  indexTypes,
		elementType: elementType,
		packed:      packed,
	}
}

func (p *program) parseSimpleType(b *block) dataType {
	if p.peek().typ == itemOpenParen {
		return p.parseEnumType(b)
	}

	lowerBound := p.parseConstant(b)

	if p.peek().typ != itemDoubleDot {
		p.errorf("expected .., got %s", p.peek())
	}
	p.next()

	upperBound := p.parseConstant(b)

	return &subrangeType{
		lowerBound: lowerBound,
		upperBound: upperBound,
	}
}

func (p *program) parseConstant(b *block) int {
	minus := false
	if p.peek().typ == itemSign {
		minus = p.next().val == "-"
	}

	if p.peek().typ == itemIdentifier {
		constantName := p.next().val
		decl := b.findConstantDeclaration(constantName)
		if decl == nil {
			p.errorf("undeclared constant %s", constantName)
		}
		v := decl.Value
		if minus {
			v = -v
		}
		return v
	}

	if p.peek().typ == itemUnsignedDigitSequence { // TODO: support all numbers
		valueStr := p.next().val
		v, err := strconv.Atoi(valueStr)
		if err != nil {
			p.errorf("literal %s is not an integer: %v", valueStr, err)
		}
		if minus {
			v = -v
		}
		return v
	}

	// TODO: support strings.

	p.errorf("got unexpected %s while parsing constant", p.peek())
	// unreachable
	return 0
}

func (p *program) parseRecordType(b *block, packed bool) *recordType {
	if p.peek().typ != itemRecord {
		p.errorf("expected record, got %s instead.", p.peek())
	}
	p.next()

	fieldList := p.parseFieldList(b)

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead.", p.peek())
	}
	p.next()

	return &recordType{
		fields: fieldList,
		packed: packed,
	}
}

func (p *program) parseFieldList(b *block) (fields []*recordField) {
	field := p.parseRecordSection(b)

	fields = append(fields, field)

	for {
		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()

		field := p.parseRecordSection(b)
		fields = append(fields, field)

	}

	return fields
}

func (p *program) parseRecordSection(b *block) *recordField {
	identifierList := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead.", p.peek())
	}
	p.next()

	typ := p.parseDataType(b)

	return &recordField{Identifiers: identifierList, Type: typ}
}

func (p *program) validateParameters(varargs bool, formalParams []*formalParameter, actualParams []expression) error {
	if varargs {
		return nil
	}

	if len(formalParams) != len(actualParams) {
		return fmt.Errorf("%d parameter(s) were declared, but %d were provided", len(formalParams), len(actualParams))
	}

	for idx := range formalParams {
		if !formalParams[idx].Type.Equals(actualParams[idx].Type()) {
			return fmt.Errorf("parameter %s expects type %s, but %s was provided",
				formalParams[idx].Name, formalParams[idx].Type.Type(), actualParams[idx].Type().Type())
		}
	}

	// TODO: check var parameters whether actual parameter is var expression.

	return nil
}
