package parser

import (
	"errors"
	"fmt"
	"runtime"
)

func parse(name, text string) (p *program, err error) {
	p = &program{
		lexer: lex(name, text),
	}
	defer p.recover(&err)
	err = p.parse()
	return p, err
}

type program struct {
	lexer     *lexer
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
	p.block = p.parseBlock()

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
	parent               *block
	labels               []string
	constantDeclarations []*constDeclaration
	typeDefinitions      []*typeDefinition
	variables            []*variable
	procedures           []*procedure
	statements           []statement
}

func (b *block) isValidLabel(label string) bool {
	for _, l := range b.labels {
		if l == label {
			return true
		}
	}
	return false
}

func (p *program) parseBlock() *block {
	b := new(block)
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
		p.errorf("expected begin, got %s intead", p.next())
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

	constDecl, ok := p.parseConstantDefinition()
	if !ok {
		p.errorf("expected constant definition")
	}
	b.constantDeclarations = append(b.constantDeclarations, constDecl)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for {
		constDecl, ok := p.parseConstantDefinition()
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
	Value string
}

func (p *program) parseConstantDefinition() (*constDeclaration, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	constName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	// TODO: support all allowed constants.
	if p.peek().typ != itemUnsignedDigitSequence {
		p.errorf("expected unsigned number, got %s", p.next())
	}
	constValue := p.next().val

	return &constDeclaration{Name: constName, Value: constValue}, true
}

func (p *program) parseTypeDeclarationPart(b *block) {
	if p.peek().typ != itemTyp {
		p.errorf("expected type, got %s", p.next())
	}
	p.next()

	b.typeDefinitions = []*typeDefinition{}
	typeDecl, ok := p.parseTypeDefinition()
	if !ok {
		p.errorf("expected type definition")
	}
	b.typeDefinitions = append(b.typeDefinitions, typeDecl)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for {
		typeDecl, ok := p.parseTypeDefinition()
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
	Type *dataType
}

func (p *program) parseTypeDefinition() (*typeDefinition, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	typeName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	dataType := p.parseDataType()

	return &typeDefinition{Name: typeName, Type: dataType}, true
}

type dataTypeType int

const (
	typeAlias dataTypeType = iota
	typePointer
	typeSubrange
	typeEnum
	typeArray
	typeRecord
	typeSet
	typeFile
)

type dataType struct {
	Type            dataTypeType
	Name            string
	EnumIdentifiers []string
	Packed          bool
	DataType        *dataType
}

func (p *program) parseDataType() *dataType {
	packed := false

restartParseDataType:
	switch p.peek().typ {
	case itemIdentifier:
		return &dataType{Type: typeAlias, Name: p.next().val}
	case itemCaret:
		p.next() // skip ^ token.
		if p.peek().typ != itemIdentifier {
			p.errorf("expected type after ^, got %s", p.next())
		}
		return &dataType{Type: typePointer, Name: p.next().val}
	case itemOpenParen:
		return &dataType{Type: typeEnum, EnumIdentifiers: p.parseEnumType()}
	case itemPacked:
		if packed {
			p.errorf("expected type after packed, got %s", p.next())
		}
		p.next()
		packed = true
		goto restartParseDataType
	case itemArray:
		return &dataType{Type: typeArray, Packed: packed}
		// TODO: parse array type
	case itemRecord:
		return &dataType{Type: typeRecord, Packed: packed}
		// TODO: parse record type
	case itemSet:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after set, got %s", p.next())
		}
		p.next()
		setDataType := p.parseDataType()
		return &dataType{Type: typeSet, Packed: packed, DataType: setDataType}
	case itemFile:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after file, got %s", p.next())
		}
		p.next()
		fileDataType := p.parseDataType()
		return &dataType{Type: typeFile, Packed: packed, DataType: fileDataType}
	default:
		p.errorf("unknown type %s", p.next().val)
	}
	// not reached.
	return nil
}

func (p *program) parseEnumType() []string {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	identifierList := p.parseIdentifierList()

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s", p.next())
	}
	p.next()

	return identifierList
}

func (p *program) parseIdentifierList() []string {
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
	Type *dataType
}

func (p *program) parseVarDeclarationPart(b *block) {
	if p.peek().typ != itemVar {
		p.errorf("expected var, got %s", p.next())
	}
	p.next()

	for {

		variableNames := p.parseIdentifierList()

		if p.peek().typ != itemColon {
			p.errorf("expected :, got %s", p.next())
		}
		p.next()

		dataType := p.parseDataType()

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
	ReturnType       *string
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
		parameterList = p.parseFormalParameterList()
	}

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	procedureBlock := p.parseBlock()
	procedureBlock.parent = b

	b.procedures = append(b.procedures, &procedure{Name: procedureName, Block: procedureBlock, FormalParameters: parameterList})
}

type formalParameter struct {
	Name           string
	Type           *dataType
	ValueParameter bool
}

func (p *program) parseFormalParameterList() []*formalParameter {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	parameterList := []*formalParameter{}

parameterListLoop:
	for {

		formalParameters := p.parseFormalParameter()

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

func (p *program) parseFormalParameter() []*formalParameter {
	valueParameter := false
	if p.peek().typ == itemVar {
		valueParameter = true
		p.next()
	}

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	parameterNames := p.parseIdentifierList()

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	parameterType := p.parseDataType()

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
		parameterList = p.parseFormalParameterList()
	}

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected type-identifier, got %s", p.next())
	}
	returnType := p.next().val

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	procedureBlock := p.parseBlock()
	procedureBlock.parent = b

	b.procedures = append(b.procedures, &procedure{Name: procedureName, Block: procedureBlock, FormalParameters: parameterList, ReturnType: &returnType})
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

type statementStruct struct {
	Type          statementType
	Label         *string
	TargetLabel   *string            // for goto
	Name          string             // for procedure call, assignment, variable in for-statement
	ParameterList []*expression      // for procedure call
	Expression    *expression        // for assignment, while, repeat until, for, if
	FinalExpr     *expression        // for final expression in for-statement
	Statements    []*statementStruct // for compound statement, repeat until
	Body          *statementStruct   // for while, for, if
	ElseBody      *statementStruct   // for if ... else
	Down          bool               // to indicate it's for ... downto ...
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
	// TODO: implement support for all types of variables.
	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	identifier := p.next().val

	switch p.peek().typ {
	case itemAssignment:
		p.next()
		expr := p.parseExpression(b)
		return &assignmentStatement{name: identifier, expr: expr}
	case itemOpenParen:
		actualParameterList := p.parseActualParameterList(b)
		return &procedureCallStatement{name: identifier, parameterList: actualParameterList}
	default:
		return &procedureCallStatement{name: identifier}
	}
}

func (p *program) parseWhileStatement(b *block) *whileStatement {
	if p.peek().typ != itemWhile {
		p.errorf("expected while, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)

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

	condition := p.parseExpression(b)

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

func (p *program) parseActualParameterList(b *block) []*expression {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	params := []*expression{}

	for {
		expr := p.parseExpression(b)

		params = append(params, expr)

		if p.peek().typ != itemSemicolon {
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

type expression struct {
	Type  exprType
	Value string
}

func (p *program) parseExpression(b *block) *expression {
	switch p.peek().typ {
	case itemIdentifier:
		return &expression{Type: exprIdentifier, Value: p.next().val}
	case itemUnsignedDigitSequence:
		return &expression{Type: exprNumber, Value: p.next().val}
	default:
		p.errorf("TODO: only identifiers and numbers allowed as expressions")
	}
	return nil
}
