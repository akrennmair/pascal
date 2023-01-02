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
	files []string
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
	err := errors.New(fmt.Sprintf("%s:%d:%d: ", p.lexer.name, p.lexer.lineNumber(), p.lexer.columnInLine()) + fmt.Sprintf(fmtstr, args...))
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

	if p.peek().typ == itemOpenParen {
		p.next()

		p.files = p.parseIdentifierList(nil)

		if p.peek().typ != itemCloseParen {
			p.errorf("expected ), got %s instead", p.peek())
		}
		p.next()
	}

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()
}

type block struct {
	parent               *block   // the parent block this block belongs to.
	routine              *routine // the procedure or function this block belongs to. nil if topmost program.
	labels               []string
	constantDeclarations []*constDeclaration
	typeDefinitions      []*typeDefinition
	variables            []*variable
	procedures           []*routine
	functions            []*routine
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

	if b.routine == nil {
		return b.parent.findFormalParameter(name)
	}

	for _, param := range b.routine.FormalParameters {
		if param.Name == name {
			return param
		}
	}

	return b.parent.findFormalParameter(name)
}

func (b *block) findProcedure(name string) *routine {
	if b == nil {
		return findBuiltinProcedure(name)
	}

	if b.routine != nil {
		for _, param := range b.routine.FormalParameters {
			if param.Name != name {
				continue
			}

			pt, ok := param.Type.(*procedureType)
			if !ok {
				continue
			}

			return &routine{
				Name:             param.Name,
				FormalParameters: pt.params,
				isParameter:      true,
			}
		}
	}

	for _, proc := range b.procedures {
		if proc.Name == name {
			return proc
		}
	}

	return b.parent.findProcedure(name)
}

func (b *block) findFunction(name string) *routine {
	if b == nil {
		return nil
	}

	if b.routine != nil {
		for _, param := range b.routine.FormalParameters {
			if param.Name != name {
				continue
			}

			ft, ok := param.Type.(*functionType)
			if !ok {
				continue
			}

			return &routine{
				Name:             param.Name,
				FormalParameters: ft.params,
				ReturnType:       ft.returnType,
				isParameter:      true,
			}
		}
	}

	for _, proc := range b.functions {
		if proc.Name == name {
			return proc
		}
	}

	return b.parent.findFunction(name)
}

func (b *block) findFunctionForAssignment(name string) *routine {
	if b == nil || b.routine == nil {
		return nil
	}

	if b.routine.Name == name {
		return b.routine
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

func (b *block) findEnumValue(ident string) (idx int, typ dataType) {
	if b == nil {
		return 0, nil
	}

	for _, v := range b.variables {
		if et, ok := v.Type.(*enumType); ok {
			for idx, enumIdent := range et.identifiers {
				if ident == enumIdent {
					return idx, v.Type
				}
			}
		}
	}

	for _, td := range b.typeDefinitions {
		if et, ok := td.Type.(*enumType); ok {
			for idx, enumIdent := range et.identifiers {
				if ident == enumIdent {
					return idx, td.Type
				}
			}
		}
	}

	return b.parent.findEnumValue(ident)
}

func (b *block) isValidLabel(label string) bool {
	for _, l := range b.labels {
		if l == label {
			return true
		}
	}
	return false
}

func (b *block) getIdentifiersInRegion() (identifiers []string) {
	identifiers = append(identifiers, b.labels...)

	for _, decl := range b.constantDeclarations {
		identifiers = append(identifiers, decl.Name)
	}

	for _, decl := range b.typeDefinitions {
		identifiers = append(identifiers, decl.Name)
	}

	for _, v := range b.variables {
		identifiers = append(identifiers, v.Name)
	}

	for _, p := range append(b.procedures, b.functions...) {
		identifiers = append(identifiers, p.Name)
	}

	return identifiers
}

func (b *block) isIdentifierUsed(name string) bool {
	for _, ident := range b.getIdentifiersInRegion() {
		if ident == name {
			return true
		}
	}
	return false
}

func (b *block) addLabel(label string) error {
	if b.isIdentifierUsed(label) {
		return fmt.Errorf("duplicate label identifier %q", label)
	}

	b.labels = append(b.labels, label)

	return nil
}

func (b *block) addConstantDefinition(constDecl *constDeclaration) error {
	if b.isIdentifierUsed(constDecl.Name) {
		return fmt.Errorf("duplicate const identifier %q", constDecl.Name)
	}

	b.constantDeclarations = append(b.constantDeclarations, constDecl)
	return nil
}

func (b *block) addTypeDefinition(typeDef *typeDefinition) error {
	if b.isIdentifierUsed(typeDef.Name) {
		return fmt.Errorf("duplicate type name %q", typeDef.Name)
	}

	b.typeDefinitions = append(b.typeDefinitions, typeDef)
	return nil
}

func (b *block) addVariable(varDecl *variable) error {
	if b.isIdentifierUsed(varDecl.Name) {
		return fmt.Errorf("duplicate variable name %q", varDecl.Name)
	}

	b.variables = append(b.variables, varDecl)
	return nil
}

func (b *block) addProcedure(proc *routine) error {
	for idx := range b.procedures {
		if b.procedures[idx].Name == proc.Name && b.procedures[idx].Forward && !proc.Forward {
			// we don't just overwrite the whole routine but instead assign the block and make it non-forward
			// as ISO Pascal allows to forward-declare procedures and functions and then
			// not have to declare the full procedure heading when actually declaring it afterwards.
			b.procedures[idx].Forward = false
			b.procedures[idx].Block = proc.Block
			return nil
		}
	}

	if b.isIdentifierUsed(proc.Name) {
		return fmt.Errorf("duplicate procedure name %q", proc.Name)
	}

	b.procedures = append(b.procedures, proc)
	return nil
}

func (b *block) addFunction(funcDecl *routine) error {
	for idx := range b.functions {
		if b.functions[idx].Name == funcDecl.Name && b.functions[idx].Forward && !funcDecl.Forward {
			// we don't just overwrite the whole routine but instead assign the block and make it non-forward
			// as ISO Pascal allows to forward-declare procedures and functions and then
			// not have to declare the full procedure heading when actually declaring it afterwards.
			b.functions[idx].Forward = false
			b.functions[idx].Block = funcDecl.Block
			return nil
		}
	}

	if b.isIdentifierUsed(funcDecl.Name) {
		return fmt.Errorf("duplicate function name %q", funcDecl.Name)
	}

	b.functions = append(b.functions, funcDecl)
	return nil
}

func (b *block) findForwardDeclaredProcedure(name string) *routine {
	for _, proc := range b.procedures {
		if proc.Name == name && proc.Forward {
			return proc
		}
	}

	return nil
}

func (b *block) findForwardDeclaredFunction(name string) *routine {
	for _, proc := range b.functions {
		if proc.Name == name && proc.Forward {
			return proc
		}
	}

	return nil
}

func (p *program) parseBlock(parent *block, proc *routine) *block {
	b := &block{
		parent:  parent,
		routine: proc,
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
		p.parseConstantDefinitionPart(b)
	}
	if p.peek().typ == itemTyp {
		p.parseTypeDefinitionPart(b)
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
		if err := b.addLabel(p.next().val); err != nil {
			p.errorf("%v", err)
		}

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

func (p *program) parseConstantDefinitionPart(b *block) {
	if p.peek().typ != itemConst {
		p.errorf("expected const, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected constant identifier, got %s instead", p.peek())
	}

	constDecl := p.parseConstantDefinition(b)
	if err := b.addConstantDefinition(constDecl); err != nil {
		p.errorf("%v", err)
	}

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for p.peek().typ == itemIdentifier {
		constDecl := p.parseConstantDefinition(b)
		if err := b.addConstantDefinition(constDecl); err != nil {
			p.errorf("%v", err)
		}

		if p.peek().typ != itemSemicolon {
			p.errorf("expected semicolon, got %s", p.next())
		}
		p.next()
	}
}

type constDeclaration struct {
	Name  string
	Value constantLiteral
}

func (p *program) parseConstantDefinition(b *block) *constDeclaration {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected constant identifier, got %s instead", p.peek())
	}

	constName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	constValue := p.parseConstant(b)

	return &constDeclaration{Name: constName, Value: constValue}
}

func (p *program) parseTypeDefinitionPart(b *block) {
	if p.peek().typ != itemTyp {
		p.errorf("expected type, got %s", p.next())
	}
	p.next()

	b.typeDefinitions = []*typeDefinition{}
	typeDef, ok := p.parseTypeDefinition(b)
	if !ok {
		p.errorf("expected type definition")
	}
	if err := b.addTypeDefinition(typeDef); err != nil {
		p.errorf("%v", err)
	}

	if p.peek().typ != itemSemicolon {
		p.errorf("expected semicolon, got %s", p.next())
	}
	p.next()

	for {
		typeDef, ok := p.parseTypeDefinition(b)
		if !ok {
			break
		}

		if err := b.addTypeDefinition(typeDef); err != nil {
			p.errorf("%v", err)
		}

		if p.peek().typ != itemSemicolon {
			p.errorf("expected semicolon, got %s", p.next())
		}
		p.next()
	}

	// resolve pointer types where the underlying type may have only been defined afterwards.
	for _, typeDef := range b.typeDefinitions {
		pt, ok := typeDef.Type.(*pointerType)
		if !ok {
			continue
		}
		if pt.name != "" && pt.typ == nil {
			pt.typ = b.findType(pt.name)
		}
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

	dataType := p.parseType(b, true)

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

type recordVariantField struct {
	tagField string
	typ      dataType
	typeName string
	variants []*recordVariant
}

type recordVariant struct {
	caseLabels []constantLiteral
	fields     *recordType
}

func (p *program) parseType(b *block, resolvePointerTypesLater bool) dataType {
	packed := false

restartParseDataType:
	switch p.peek().typ {
	case itemIdentifier:
		ident := p.peek().val
		// if identifier is an already existing type name, it's an alias.
		if typ := b.findType(ident); typ != nil {
			p.next()
			return typ
		}

		// if the identifier is an already existing constant, it can only be a constant being used in a subrange type.
		if constDecl := b.findConstantDeclaration(ident); constDecl != nil {
			return p.parseSubrangeType(b)
		} else if _, typ := b.findEnumValue(ident); typ != nil {
			return p.parseSubrangeType(b)
		}

		// otherwise, we don't know.
		p.errorf("unknown type %s", ident)
	case itemCaret:
		p.next() // skip ^ token.
		if p.peek().typ != itemIdentifier {
			p.errorf("expected type after ^, got %s", p.next())
		}

		ident := p.next().val

		typeDecl := b.findType(ident)
		if typeDecl == nil {
			if resolvePointerTypesLater {
				return &pointerType{name: ident}
			}
			p.errorf("unknown type %s", ident)
		}

		return &pointerType{typ: typeDecl}
	case itemOpenParen:
		return p.parseEnumType(b)
	case itemPacked:
		p.next()
		packed = true
		if typ := p.peek().typ; typ != itemArray && typ != itemRecord && typ != itemSet && typ != itemFile {
			p.errorf("packed can only be used with array, record, set or file, found %s instead", p.peek())
		}
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
		setDataType := p.parseType(b, resolvePointerTypesLater)
		return &setType{elementType: setDataType, packed: packed}
	case itemFile:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after file, got %s", p.next())
		}
		p.next()
		fileDataType := p.parseType(b, resolvePointerTypesLater)
		return &fileType{elementType: fileDataType, packed: packed}
	case itemSign, itemUnsignedDigitSequence:
		// if the type definition is a sign or digits, it can only be a subrange type.
		return p.parseSubrangeType(b)
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

	// TODO: ensure that identifiers in identifier list are indeed unique.

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

	// the following fields are only set for variables that are looked up from within with statements,
	// and they indicate that Name and Type describe the field of a record variable of name BelongsTo of
	// type BelongsToType.
	BelongsTo     string
	BelongsToType dataType
}

func (p *program) parseVarDeclarationPart(b *block) {
	if p.peek().typ != itemVar {
		p.errorf("expected var, got %s", p.next())
	}
	p.next()

	for {
		p.parseVariableDeclaration(b)

		if p.peek().typ != itemIdentifier {
			break
		}
	}
}

func (p *program) parseVariableDeclaration(b *block) {
	variableNames := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	dataType := p.parseType(b, false)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	for _, varName := range variableNames {
		if err := b.addVariable(&variable{Name: varName, Type: dataType}); err != nil {
			p.errorf("%v", err)
		}
	}
}

func (p *program) parseProcedureAndFunctionDeclarationPart(b *block) {
	for {
		switch p.peek().typ {
		case itemProcedure:
			p.parseProcedureDeclaration(b)
		case itemFunction:
			p.parseFunctionDeclaration(b)
		default:
			return
		}
		if p.peek().typ != itemSemicolon {
			p.errorf("expected ;, got %s", p.next())
		}
		p.next()
	}
}

type routine struct {
	Name             string
	Block            *block
	FormalParameters []*formalParameter
	ReturnType       dataType
	Forward          bool // if true, routine is only forward-declared.
	varargs          bool // for builtin functions with variable arguments.
	isParameter      bool // if true, indicates that this refers to a procedural or functional parameter
}

func (p *program) parseProcedureDeclaration(b *block) {
	procedureName, parameterList := p.parseProcedureHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &routine{Name: procedureName, FormalParameters: parameterList}

	if p.peek().typ == itemForward {
		p.next()
		proc.Forward = true
	} else {
		procForParsing := proc
		if fwdProc := b.findForwardDeclaredProcedure(procedureName); fwdProc != nil {
			procForParsing = fwdProc
		}
		proc.Block = p.parseBlock(b, procForParsing)
	}

	if err := b.addProcedure(proc); err != nil {
		p.errorf("%v", err)
	}
}

func (p *program) parseProcedureHeading(b *block) (string, []*formalParameter) {
	if p.peek().typ != itemProcedure {
		p.errorf("expected procedure, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected procedure identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var parameterList []*formalParameter
	if p.peek().typ == itemOpenParen {
		parameterList = p.parseFormalParameterList(b)
	}
	return procedureName, parameterList
}

type formalParameter struct {
	Name              string
	Type              dataType
	VariableParameter bool
}

func (p *formalParameter) String() string {
	var buf strings.Builder

	if p.VariableParameter {
		buf.WriteString("var ")
	}

	switch p.Type.(type) {
	case *functionType:
		buf.WriteString("function ")
		buf.WriteString(p.Name)
		buf.WriteString(p.Type.Type())
	case *procedureType:
		buf.WriteString("procedure ")
		buf.WriteString(p.Name)
		buf.WriteString(p.Type.Type())
	default:
		buf.WriteString(p.Name)
		buf.WriteString(" : ")
		buf.WriteString(p.Type.Type())
	}

	return buf.String()
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
	variableParam := false

	var formalParameters []*formalParameter

	switch p.peek().typ {
	case itemProcedure:
		p.next()

		if p.peek().typ != itemIdentifier {
			p.errorf("expected procedure name, got %s instead", p.peek())
		}

		name := p.next().val

		params := p.parseFormalParameterList(b)

		formalParameters = append(formalParameters, &formalParameter{
			Name: name,
			Type: &procedureType{params: params},
		})
	case itemFunction:
		p.next()

		if p.peek().typ != itemIdentifier {
			p.errorf("expected function name, got %s instead", p.peek())
		}

		name := p.next().val

		params := p.parseFormalParameterList(b)

		if p.peek().typ != itemColon {
			p.errorf("expected : after formal parameter list, got %s instead", p.peek())
		}

		p.next()

		returnType := p.parseType(b, false)

		formalParameters = append(formalParameters, &formalParameter{
			Name: name,
			Type: &functionType{params: params, returnType: returnType},
		})
	case itemVar:
		variableParam = true
		p.next()
		fallthrough
	case itemIdentifier:
		parameterNames := p.parseIdentifierList(b)

		if p.peek().typ != itemColon {
			p.errorf("expected :, got %s", p.next())
		}
		p.next()

		parameterType := p.parseType(b, false)

		formalParameters = make([]*formalParameter, 0, len(parameterNames))
		for _, name := range parameterNames {
			formalParameters = append(formalParameters, &formalParameter{Name: name, Type: parameterType, VariableParameter: variableParam})
		}
	default:
		p.errorf("expected var, procedure, function or identifier, got %s", p.peek())
	}

	return formalParameters
}

func (p *program) parseFunctionDeclaration(b *block) {
	funcName, parameterList, returnType := p.parseFunctionHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &routine{Name: funcName, FormalParameters: parameterList, ReturnType: returnType}

	if p.peek().typ == itemForward {
		p.next()
		proc.Forward = true
	} else {
		procForParsing := proc
		if fwdProc := b.findForwardDeclaredFunction(funcName); fwdProc != nil {
			procForParsing = fwdProc
		}
		proc.Block = p.parseBlock(b, procForParsing)
	}

	if err := b.addFunction(proc); err != nil {
		p.errorf("%v", err)
	}
}

func (p *program) parseFunctionHeading(b *block) (string, []*formalParameter, dataType) {
	if p.peek().typ != itemFunction {
		p.errorf("expected function, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected function identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var (
		parameterList []*formalParameter
		returnType    dataType
	)

	if p.peek().typ == itemOpenParen {
		parameterList = p.parseFormalParameterList(b)
	}

	if p.peek().typ == itemColon {
		p.next()
		returnType = p.parseType(b, false)
	}

	return procedureName, parameterList, returnType
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
	stmtCase
	stmtWith
)

type statement interface {
	Type() statementType
	Label() *string
}

func (p *program) parseStatement(b *block) statement {
	var label string
	if p.peek().typ == itemUnsignedDigitSequence {
		label = p.next().val

		if !b.isValidLabel(label) {
			p.errorf("undeclared label %s", label)
		}

		if p.peek().typ != itemColon {
			p.errorf("expected : after label, got %s", p.next())
		}
		p.next()
	}

	stmt := p.parseUnlabelledStatement(b)
	if label != "" {
		stmt = &labelledStatement{label: label, statement: stmt}
	}

	return stmt
}

func (p *program) parseUnlabelledStatement(b *block) statement {
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
		return &statementGoto{target: tl}
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
	case itemWith:
		return p.parseWithStatement(b)
	}
	p.errorf("unsupported %s as statement", p.next())
	return nil
}

func (p *program) parseAssignmentOrProcedureStatement(b *block) statement {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	identifier := p.next().val

	if p.peek().typ == itemOpenParen {
		proc := b.findProcedure(identifier)
		if proc == nil {
			p.errorf("unknown procedure %s", identifier)
		}
		actualParameterList := p.parseActualParameterList(b)
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, actualParameterList); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &procedureCallStatement{name: identifier, parameterList: actualParameterList}
	}

	proc := b.findProcedure(identifier)
	if proc != nil {
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, []expression{}); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &procedureCallStatement{name: identifier}
	}

	var lexpr expression

	if funcDecl := b.findFunctionForAssignment(identifier); funcDecl != nil {
		lexpr = &variableExpr{name: identifier, typ: funcDecl.ReturnType} // TODO: do we need a separate expression type for this?
	} else {
		lexpr = p.parseVariable(b, identifier)
	}

	if p.peek().typ == itemAssignment {
		p.next()
		if lexpr == nil {
			p.errorf("assignment: unknown left expression %s", identifier)
		}
		rexpr := p.parseExpression(b)
		if !lexpr.Type().Equals(rexpr.Type()) {
			p.errorf("incompatible types: got %s, expected %s", rexpr.Type().Type(), lexpr.Type().Type())
		}
		return &assignmentStatement{lexpr: lexpr, rexpr: rexpr}
	}

	p.errorf("unexpected token %s in statement", p.peek())
	// unreachable
	return nil
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
		p.errorf("unknown variable %s in for statement", variable)
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
	if p.peek().typ != itemCase {
		p.errorf("expected case, got %s instead", p.peek())
	}
	p.next()

	expr := p.parseExpression(b)

	if p.peek().typ != itemOf {
		p.errorf("expected of, got %s instead", p.peek())
	}
	p.next()

	var caseLimbs []*caseLimb

	limb := p.parseCaseLimb(b)
	for _, label := range limb.labels {
		if !expr.Type().Equals(label.ConstantType()) {
			p.errorf("case label %s doesn't match case expression type %s", label.String(), expr.Type().Type())
		}
	}
	caseLimbs = append(caseLimbs, limb)

	for {
		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()

		if !isPossiblyConstant(b, p.peek()) {
			break
		}

		limb := p.parseCaseLimb(b)
		for _, label := range limb.labels {
			if !expr.Type().Equals(label.ConstantType()) {
				p.errorf("case label %s doesn't match case expression type %s", label.String(), expr.Type().Type())
			}
		}
		caseLimbs = append(caseLimbs, limb)
	}

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead", p.peek())
	}
	p.next()

	return &caseStatement{expr: expr, caseLimbs: caseLimbs}
}

func (p *program) parseCaseLimb(b *block) *caseLimb {
	labels := p.parseCaseLabelList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead", p.peek())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &caseLimb{
		labels: labels,
		stmt:   stmt,
	}
}

func (p *program) parseWithStatement(b *block) statement {
	if p.peek().typ != itemWith {
		p.errorf("expected with, got %s instead", p.peek())
	}
	p.next()

	withBlock := &block{
		parent:  b,
		routine: b.routine,
	}

	var recordVariables []string

	for {

		if p.peek().typ != itemIdentifier {
			p.errorf("expected identifier of record variable, got %s instead", p.peek())
		}
		ident := p.next().val

		var typ dataType

		if varDecl := b.findVariable(ident); varDecl != nil {
			typ = varDecl.Type
		} else if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
			typ = paramDecl.Type
		} else {
			p.errorf("unknown variable %s x", ident)
		}

		recType, ok := typ.(*recordType)
		if !ok {
			p.errorf("variable %s is not a record variable", ident)
		}

		recordVariables = append(recordVariables, ident)

		for _, field := range recType.fields {
			for _, fieldIdent := range field.Identifiers {
				fieldVar := &variable{
					Name:          fieldIdent,
					Type:          field.Type,
					BelongsTo:     ident,
					BelongsToType: recType,
				}

				withBlock.variables = append(withBlock.variables, fieldVar)
			}
		}

		if p.peek().typ != itemComma {
			break
		}
	}

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s instead", p.peek())
	}
	p.next()

	stmt := p.parseStatement(withBlock)

	withBlock.statements = append(withBlock.statements, stmt)

	return &withStatement{
		recordVariables: recordVariables,
		block:           withBlock,
	}
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
			p.errorf("in: expected set type, got %s instead.", rt.Type())
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
			if !isIntegerType(simpleExpr.first.Type()) && !isRealType(simpleExpr.first.Type()) {
				p.errorf("can only use %s operator with integer or real types, got %s instead", operator, simpleExpr.first.Type().Type())
			}
		}

		nextTerm := p.parseTerm(b)

		if !typesCompatible(simpleExpr.first.Type(), nextTerm.Type()) {
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

		switch operator {
		case opAnd:
			_, ok := term.first.Type().(*booleanType)
			if !ok {
				p.errorf("can't use and with %s", term.first.Type().Type())
			}
		case opMultiply:
			if !isIntegerType(term.first.Type()) && !isRealType(term.first.Type()) {
				p.errorf("can only use %s operator with integer or real types, got %s instead", operator, term.first.Type().Type())
			}
		case opFloatDivide:
			if !isRealType(term.first.Type()) {
				p.errorf("can only use %s operator with real types, got %s instead", operator, term.first.Type().Type())
			}
		case opDivide, opModulo:
			if !isIntegerType(term.first.Type()) {
				p.errorf("can only use %s operator with integer types, got %s intead", operator, term.first.Type().Type())
			}
		}

		nextFactor := p.parseFactor(b)

		if !typesCompatible(term.first.Type(), nextFactor.Type()) {
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

		if funcDecl := b.findFunction(ident); funcDecl != nil {
			if p.peek().typ == itemOpenParen {
				params := p.parseActualParameterList(b)
				if err := p.validateParameters(funcDecl.varargs, funcDecl.FormalParameters, params); err != nil {
					p.errorf("function %s: %v", ident, err)
				}
				return &functionCallExpr{name: ident, params: params, typ: funcDecl.ReturnType}
			}

			if len(funcDecl.FormalParameters) > 0 { // function has formal parameter which are not provided -> it's a functional-parameter
				return &variableExpr{name: ident, typ: &functionType{params: funcDecl.FormalParameters, returnType: funcDecl.ReturnType}}
			}
			// TODO: what if function has no formal parameters? is it a functional parameter or a function call? needs resolved later, probably.
			if err := p.validateParameters(funcDecl.varargs, funcDecl.FormalParameters, []expression{}); err != nil {
				p.errorf("function %s: %v", ident, err)
			}
			return &functionCallExpr{name: ident, typ: funcDecl.ReturnType}

		}
		if constDecl := b.findConstantDeclaration(ident); constDecl != nil {
			return &constantExpr{ident, constDecl.Value.ConstantType()}
		}
		if idx, typ := b.findEnumValue(ident); typ != nil {
			return &enumValueExpr{symbol: ident, value: idx, typ: typ}
		}

		return p.parseVariable(b, ident)
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

func (p *program) parseVariable(b *block, ident string) expression {
	var expr expression

	if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
		expr = &variableExpr{name: ident, typ: paramDecl.Type} // TODO: do we need a separate formal parameter expression here?
	} else if procDecl := b.findProcedure(ident); procDecl != nil { // TODO: do we need a separate procedural parameter expression here?
		expr = &variableExpr{name: ident, typ: &procedureType{params: procDecl.FormalParameters}}
	} else if varDecl := b.findVariable(ident); varDecl != nil {
		expr = &variableExpr{name: ident, typ: varDecl.Type}
	}

	if expr == nil {
		p.errorf("unknown identifier %s", ident)
	}

	cont := true

	for cont {
		switch p.peek().typ {
		case itemCaret:
			_, ok := expr.Type().(*pointerType)
			if !ok {
				p.errorf("attempting to ^ but expression is not a pointer type")
			}
			p.next()
			expr = &derefExpr{expr: expr}
		case itemOpenBracket:
			expr = p.parseIndexVariableExpr(b, expr)
		case itemDot:
			p.next()

			rt, ok := expr.Type().(*recordType)
			if !ok {
				p.errorf("expression is a record type")
			}

			if p.peek().typ != itemIdentifier {
				p.errorf("expected identifier, got %s instead", p.peek())
			}
			fieldIdentifier := p.next().val
			field := rt.findField(fieldIdentifier)

			expr = &fieldDesignatorExpr{expr: expr, field: fieldIdentifier, typ: field.Type}
		default:
			cont = false
		}
	}

	return expr
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

	elementType := p.parseType(b, false)

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

	return p.parseSubrangeType(b)
}

func (p *program) parseSubrangeType(b *block) dataType {
	lowerBound := p.parseConstant(b)
	var (
		lowerValue int
		upperValue int
		typ        dataType
		upperType  dataType
	)

	switch lb := lowerBound.(type) {
	case *integerLiteral:
		lowerValue = lb.Value
		typ = lb.ConstantType()
	case *enumValueLiteral:
		lowerValue = lb.Value
		typ = lb.Type
	default:
		p.errorf("expected lower bound to be an integer or an enum value, got a %s instead", lb.ConstantType().Type())
	}

	if p.peek().typ != itemDoubleDot {
		p.errorf("expected .., got %s", p.peek())
	}
	p.next()

	upperBound := p.parseConstant(b)

	switch ub := upperBound.(type) {
	case *integerLiteral:
		upperValue = ub.Value
		upperType = ub.ConstantType()
	case *enumValueLiteral:
		upperValue = ub.Value
		upperType = ub.Type
	default:
		p.errorf("expected upper bound to be an integer or an enum value, got a %s instead", ub.ConstantType().Type())
	}

	if !upperType.Equals(typ) {
		p.errorf("type of lower bound differs from upper bound: %s vs %s", typ.Type(), upperType.Type())
	}

	return &subrangeType{
		lowerBound: lowerValue,
		upperBound: upperValue,
		typ:        typ,
	}
}

func (p *program) parseConstant(b *block) constantLiteral {
	minus := false
	if p.peek().typ == itemSign {
		minus = p.next().val == "-"
	}
	return p.parseConstantWithoutSign(b, minus)
}

func isPossiblyConstant(b *block, it item) bool {
	if it.typ == itemIdentifier {
		if b.findConstantDeclaration(it.val) != nil {
			return true
		}

		if _, typ := b.findEnumValue(it.val); typ == nil {
			return false
		}

		return true
	}

	return it.typ == itemSign || it.typ == itemUnsignedDigitSequence || it.typ == itemStringLiteral
}

func (p *program) parseConstantWithoutSign(b *block, minus bool) constantLiteral {
	var v constantLiteral

	if p.peek().typ == itemIdentifier {
		constantName := p.next().val
		decl := b.findConstantDeclaration(constantName)
		if decl != nil {
			v = decl.Value
		} else {
			idx, typ := b.findEnumValue(constantName)
			if typ == nil {
				p.errorf("undeclared constant %s", constantName)
			}

			v = &enumValueLiteral{Symbol: constantName, Value: idx, Type: typ}
		}
	} else if p.peek().typ == itemUnsignedDigitSequence {
		number := p.parseNumber(false) // negation will be done later on.
		switch n := number.(type) {
		case *integerExpr:
			v = &integerLiteral{Value: int(n.val)}
		case *floatExpr:
			v = &floatLiteral{minus: n.minus, beforeComma: n.beforeComma, afterComma: n.afterComma, scaleFactor: n.scaleFactor}
		}
	} else if p.peek().typ == itemStringLiteral {
		v = &stringLiteral{Value: p.next().val}
	} else {
		p.errorf("got unexpected %s while parsing constant", p.peek())
	}

	if minus {
		nv, err := v.Negate()
		if err != nil {
			p.errorf("%v", err)
		}
		v = nv
	}

	return v
}

func (p *program) parseRecordType(b *block, packed bool) *recordType {
	if p.peek().typ != itemRecord {
		p.errorf("expected record, got %s instead.", p.peek())
	}
	p.next()

	record := p.parseFieldList(b, packed)

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead.", p.peek())
	}
	p.next()

	return record
}

func (p *program) parseFieldList(b *block, packed bool) *recordType {
	record := &recordType{packed: packed}

	if p.peek().typ != itemCase && p.peek().typ != itemIdentifier {
		// if it's neither a case nor an identifier, we probably have an empty field list
		return record
	}

	if p.peek().typ != itemCase {
		record.fields = p.parsedFixedPart(b)
	}

	if p.peek().typ == itemCase {
		record.variantField = p.parseVariantField(b, packed)
	}

	if p.peek().typ == itemSemicolon {
		p.next()
	}

	fieldNames := map[string]bool{}

	for _, f := range record.fields {
		for _, ident := range f.Identifiers {
			if fieldNames[ident] {
				p.errorf("duplicate field name %s", ident)
			}
			fieldNames[ident] = true
		}
	}

	if record.variantField != nil {
		for _, v := range record.variantField.variants {
			for _, f := range v.fields.fields {
				for _, ident := range f.Identifiers {
					if fieldNames[ident] {
						p.errorf("duplicate variant field name %s", ident)
					}
					fieldNames[ident] = true
				}
			}
		}
	}

	return record
}

func (p *program) parsedFixedPart(b *block) (fields []*recordField) {
	field := p.parseRecordSection(b)
	fields = append(fields, field)

	for {
		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()

		if p.peek().typ != itemIdentifier {
			break
		}

		field := p.parseRecordSection(b)
		fields = append(fields, field)
	}

	return fields
}

func (p *program) parseVariantField(b *block, packed bool) (field *recordVariantField) {
	if p.peek().typ != itemCase {
		p.errorf("expected case, got %s instead", p.peek())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s instead", p.peek())
	}

	typeIdentifier := p.next().val

	tag := ""

	if p.peek().typ == itemColon {
		p.next()
		tag = typeIdentifier
		if p.peek().typ != itemIdentifier {
			p.errorf("expected identifier, got %s instead", p.peek())
		}
		typeIdentifier = p.next().val
	}

	typeDef := b.findType(typeIdentifier)
	if typeDef == nil {
		p.errorf("unknown type identifier %s", typeIdentifier)
	}

	if p.peek().typ != itemOf {
		p.errorf("expected of, got %s instead", p.peek())
	}
	p.next()

	field = &recordVariantField{
		tagField: tag,
		typ:      typeDef,
		typeName: typeIdentifier,
	}

	for {
		if !isPossiblyConstant(b, p.peek()) { // variant always starts with constant.
			break
		}

		variant := p.parseVariant(b, packed)
		field.variants = append(field.variants, variant)

		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()
	}

	return field
}

func (p *program) parseVariant(b *block, packed bool) *recordVariant {
	labels := p.parseCaseLabelList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead", p.peek())
	}
	p.next()

	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s instead", p.peek())
	}
	p.next()

	fieldList := p.parseFieldList(b, packed)

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s instead", p.peek())
	}
	p.next()

	return &recordVariant{
		caseLabels: labels,
		fields:     fieldList,
	}
}

func (p *program) parseCaseLabelList(b *block) (labels []constantLiteral) {
	label := p.parseConstant(b)
	labels = append(labels, label)

	for p.peek().typ == itemComma {
		p.next()

		label := p.parseConstant(b)
		labels = append(labels, label)
	}

	return labels
}

func (p *program) parseRecordSection(b *block) *recordField {
	identifierList := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead.", p.peek())
	}
	p.next()

	typ := p.parseType(b, false)

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

		if formalParams[idx].VariableParameter {
			if !actualParams[idx].IsVariableExpr() {
				return fmt.Errorf("parameter %s is a variable parameter, but an actual parameter other than variable was provided",
					formalParams[idx].Name)
			}
		}
	}

	return nil
}

func (p *program) parseIndexVariableExpr(b *block, expr expression) *indexedVariableExpr {
	p.next()
	indexes := p.parseExpressionList(b)
	if p.peek().typ != itemCloseBracket {
		p.errorf("expected ], got %s instead", p.peek())
	}
	p.next()

	arrType, ok := expr.Type().(*arrayType)
	if !ok {
		p.errorf("expression is not array")
	}

	// TODO: support situation where fewer index expressions mean that an array of fewer dimensions is returned.

	if len(arrType.indexTypes) != len(indexes) {
		p.errorf("array has %d dimensions but %d index expressions were provided", len(arrType.indexTypes), len(indexes))
	}

	for idx, idxType := range arrType.indexTypes {
		if !typesCompatible(idxType, indexes[idx].Type()) {
			p.errorf("array dimension %d is of type %s, but index expression type %s was provided", idx, idxType.Type(), indexes[idx].Type().Type())
		}
	}

	return &indexedVariableExpr{expr: expr, exprs: indexes, typ: arrType.elementType}
}
