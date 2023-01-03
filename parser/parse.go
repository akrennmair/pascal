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

func newParser(name, text string) *parser {
	return &parser{
		lexer:  lex(name, text),
		logger: log.New(io.Discard, "parser", log.LstdFlags|log.Lshortfile),
	}
}

func (p *parser) setLogOutput(w io.Writer) {
	p.logger.SetOutput(w)
}

func (p *parser) Parse() (ast *AST, err error) {
	defer p.recover(&err)
	ast, err = p.parse()
	return ast, err
}

// Parse parses a single Pascal file, identified by name. The
// file content must be provided in text. It returns the
// Abstract Syntax Tree (AST) as a *AST object, or an error.
func Parse(name, text string) (ast *AST, err error) {
	ast, err = parseWithLexer(lex(name, text))
	return ast, err
}

func parseWithLexer(lexer *lexer) (ast *AST, err error) {
	p := &parser{
		lexer:  lexer,
		logger: log.New(io.Discard, "parser", log.LstdFlags|log.Lshortfile),
	}
	defer p.recover(&err)
	ast, err = p.parse()
	return ast, err
}

type parser struct {
	lexer     *lexer
	logger    *log.Logger
	token     [3]item
	peekCount int
}

type AST struct {
	// The program name.
	Name string

	// The Files provided in the program heading.
	Files []string

	// The top-most block of the program that contains all global
	// declarations and definitions as well as the main program
	// to be executed.
	Block *Block
}

func (p *parser) recover(errp *error) {
	e := recover()
	if e != nil {
		// rethrow runtime errors
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
}

func (p *parser) backup() {
	p.peekCount++
}

func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lexer.nextItem()
	return p.token[0]
}

func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lexer.nextItem()
	}
	i := p.token[p.peekCount]
	return i
}

func (p *parser) errorf(fmtstr string, args ...interface{}) {
	err := errors.New(fmt.Sprintf("%s:%d:%d: ", p.lexer.name, p.lexer.lineNumber(), p.lexer.columnInLine()) + fmt.Sprintf(fmtstr, args...))
	panic(err)
}

func (p *parser) parse() (ast *AST, err error) {
	defer p.recover(&err)

	ast = &AST{}

	p.parseProgramHeading(ast)
	ast.Block = p.parseBlock(nil, nil)

	if p.peek().typ != itemDot {
		p.errorf("expected ., got %s instead", p.next())
	}
	p.next()

	return ast, nil
}

func (p *parser) parseProgramHeading(ast *AST) {
	if p.peek().typ != itemProgram {
		p.errorf("expected program, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}
	ast.Name = p.next().val

	if p.peek().typ == itemOpenParen {
		p.next()

		ast.Files = p.parseIdentifierList(nil)

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

// Block describes a program block. A program block consists of
// declarations (label declarations, constant definitions, type definitions,
// variable declarations and procedure and function delcarations) and
// the statements associated with the block.
type Block struct {
	// Parent points to the block this block belongs to, e.g. a procedure block
	// declared at the top level points to the program's main block. The program's
	// main block has no parent.
	Parent *Block

	// Routine points to the Routine (procedure or function) this block belongs to.
	// This field is nil if the block is the program's main block.
	Routine *Routine

	// Labels contains all declared Labels in the block. All Labels are unsigned digit sequences.
	Labels []string

	// Constants contains all constant definitions in the block.
	Constants []*ConstantDefinition

	// Types contains all type definitions in the block.
	Types []*TypeDefinition

	// Variables contains all variables declared in the block.
	Variables []*Variable

	// Procedure contains all procedures declared in the block.
	Procedures []*Routine

	// Functions contains all functions declared in the block.
	Functions []*Routine

	// Statements contains all Statements declared in the block.
	Statements []Statement
}

func (b *Block) findConstantDeclaration(name string) *ConstantDefinition {
	if b == nil {
		return nil
	}

	for _, constant := range b.Constants {
		if constant.Name == name {
			return constant
		}
	}

	return b.Parent.findConstantDeclaration(name)
}

func (b *Block) findVariable(name string) *Variable {
	if b == nil {
		return nil
	}

	for _, variable := range b.Variables {
		if variable.Name == name {
			return variable
		}
	}

	return b.Parent.findVariable(name)
}

func (b *Block) findFormalParameter(name string) *FormalParameter {
	if b == nil {
		return nil
	}

	if b.Routine == nil {
		return b.Parent.findFormalParameter(name)
	}

	for _, param := range b.Routine.FormalParameters {
		if param.Name == name {
			return param
		}
	}

	return b.Parent.findFormalParameter(name)
}

func (b *Block) findProcedure(name string) *Routine {
	if b == nil {
		return findBuiltinProcedure(name)
	}

	if b.Routine != nil {
		for _, param := range b.Routine.FormalParameters {
			if param.Name != name {
				continue
			}

			pt, ok := param.Type.(*ProcedureType)
			if !ok {
				continue
			}

			return &Routine{
				Name:             param.Name,
				FormalParameters: pt.FormalParams,
				isParameter:      true,
			}
		}
	}

	for _, proc := range b.Procedures {
		if proc.Name == name {
			return proc
		}
	}

	return b.Parent.findProcedure(name)
}

func (b *Block) findFunction(name string) *Routine {
	if b == nil {
		return nil
	}

	if b.Routine != nil {
		for _, param := range b.Routine.FormalParameters {
			if param.Name != name {
				continue
			}

			ft, ok := param.Type.(*FunctionType)
			if !ok {
				continue
			}

			return &Routine{
				Name:             param.Name,
				FormalParameters: ft.FormalParams,
				ReturnType:       ft.ReturnType,
				isParameter:      true,
			}
		}
	}

	for _, proc := range b.Functions {
		if proc.Name == name {
			return proc
		}
	}

	return b.Parent.findFunction(name)
}

func (b *Block) findFunctionForAssignment(name string) *Routine {
	if b == nil || b.Routine == nil {
		return nil
	}

	if b.Routine.Name == name {
		return b.Routine
	}

	return b.Parent.findFunctionForAssignment(name)
}

func (b *Block) findType(name string) DataType {
	typ := getBuiltinType(name)
	if typ != nil {
		return typ
	}

	if b == nil {
		return nil
	}

	var foundType DataType

	for _, typ := range b.Types {
		if typ.Name == name {
			foundType = typ.Type
			break
		}
	}

	if foundType == nil {
		foundType = b.Parent.findType(name)
	}

	return foundType
}

func (b *Block) findEnumValue(ident string) (idx int, typ DataType) {
	if b == nil {
		return 0, nil
	}

	for _, v := range b.Variables {
		if et, ok := v.Type.(*EnumType); ok {
			for idx, enumIdent := range et.Identifiers {
				if ident == enumIdent {
					return idx, v.Type
				}
			}
		}
	}

	for _, td := range b.Types {
		if et, ok := td.Type.(*EnumType); ok {
			for idx, enumIdent := range et.Identifiers {
				if ident == enumIdent {
					return idx, td.Type
				}
			}
		}
	}

	return b.Parent.findEnumValue(ident)
}

func (b *Block) isValidLabel(label string) bool {
	for _, l := range b.Labels {
		if l == label {
			return true
		}
	}
	return false
}

func (b *Block) getIdentifiersInRegion() (identifiers []string) {
	identifiers = append(identifiers, b.Labels...)

	for _, decl := range b.Constants {
		identifiers = append(identifiers, decl.Name)
	}

	for _, decl := range b.Types {
		identifiers = append(identifiers, decl.Name)
	}

	for _, v := range b.Variables {
		identifiers = append(identifiers, v.Name)
	}

	for _, p := range append(b.Procedures, b.Functions...) {
		identifiers = append(identifiers, p.Name)
	}

	return identifiers
}

func (b *Block) isIdentifierUsed(name string) bool {
	for _, ident := range b.getIdentifiersInRegion() {
		if ident == name {
			return true
		}
	}
	return false
}

func (b *Block) addLabel(label string) error {
	if b.isIdentifierUsed(label) {
		return fmt.Errorf("duplicate label identifier %q", label)
	}

	b.Labels = append(b.Labels, label)

	return nil
}

func (b *Block) addConstantDefinition(constDecl *ConstantDefinition) error {
	if b.isIdentifierUsed(constDecl.Name) {
		return fmt.Errorf("duplicate const identifier %q", constDecl.Name)
	}

	b.Constants = append(b.Constants, constDecl)
	return nil
}

func (b *Block) addTypeDefinition(typeDef *TypeDefinition) error {
	if b.isIdentifierUsed(typeDef.Name) {
		return fmt.Errorf("duplicate type name %q", typeDef.Name)
	}

	b.Types = append(b.Types, typeDef)
	return nil
}

func (b *Block) addVariable(varDecl *Variable) error {
	if b.isIdentifierUsed(varDecl.Name) {
		return fmt.Errorf("duplicate variable name %q", varDecl.Name)
	}

	b.Variables = append(b.Variables, varDecl)
	return nil
}

func (b *Block) addProcedure(proc *Routine) error {
	for idx := range b.Procedures {
		if b.Procedures[idx].Name == proc.Name && b.Procedures[idx].Forward && !proc.Forward {
			// we don't just overwrite the whole routine but instead assign the block and make it non-forward
			// as ISO Pascal allows to forward-declare procedures and functions and then
			// not have to declare the full procedure heading when actually declaring it afterwards.
			b.Procedures[idx].Forward = false
			b.Procedures[idx].Block = proc.Block
			return nil
		}
	}

	if b.isIdentifierUsed(proc.Name) {
		return fmt.Errorf("duplicate procedure name %q", proc.Name)
	}

	b.Procedures = append(b.Procedures, proc)
	return nil
}

func (b *Block) addFunction(funcDecl *Routine) error {
	for idx := range b.Functions {
		if b.Functions[idx].Name == funcDecl.Name && b.Functions[idx].Forward && !funcDecl.Forward {
			// we don't just overwrite the whole routine but instead assign the block and make it non-forward
			// as ISO Pascal allows to forward-declare procedures and functions and then
			// not have to declare the full procedure heading when actually declaring it afterwards.
			b.Functions[idx].Forward = false
			b.Functions[idx].Block = funcDecl.Block
			return nil
		}
	}

	if b.isIdentifierUsed(funcDecl.Name) {
		return fmt.Errorf("duplicate function name %q", funcDecl.Name)
	}

	b.Functions = append(b.Functions, funcDecl)
	return nil
}

func (b *Block) findForwardDeclaredProcedure(name string) *Routine {
	for _, proc := range b.Procedures {
		if proc.Name == name && proc.Forward {
			return proc
		}
	}

	return nil
}

func (b *Block) findForwardDeclaredFunction(name string) *Routine {
	for _, proc := range b.Functions {
		if proc.Name == name && proc.Forward {
			return proc
		}
	}

	return nil
}

func (p *parser) parseBlock(parent *Block, proc *Routine) *Block {
	b := &Block{
		Parent:  parent,
		Routine: proc,
	}
	p.parseDeclarationPart(b)
	p.parseStatementPart(b)
	return b
}

func (p *parser) parseDeclarationPart(b *Block) {
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

func (p *parser) parseStatementPart(b *Block) {
	if p.peek().typ != itemBegin {
		p.errorf("expected begin, got %s instead", p.next())
	}
	p.next()

	b.Statements = p.parseStatementSequence(b)

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead", p.next())
	}
	p.next()
}

func (p *parser) parseLabelDeclarationPart(b *Block) {
	if p.peek().typ != itemLabel {
		p.errorf("expected label, got %s", p.next())
	}
	p.next()

	b.Labels = []string{}

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

func (p *parser) parseConstantDefinitionPart(b *Block) {
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

type ConstantDefinition struct {
	Name  string
	Value ConstantLiteral
}

func (p *parser) parseConstantDefinition(b *Block) *ConstantDefinition {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected constant identifier, got %s instead", p.peek())
	}

	constName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	constValue := p.parseConstant(b)

	return &ConstantDefinition{Name: constName, Value: constValue}
}

func (p *parser) parseTypeDefinitionPart(b *Block) {
	if p.peek().typ != itemTyp {
		p.errorf("expected type, got %s", p.next())
	}
	p.next()

	b.Types = []*TypeDefinition{}
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
	for _, typeDef := range b.Types {
		pt, ok := typeDef.Type.(*PointerType)
		if !ok {
			continue
		}
		if pt.Name != "" && pt.Type_ == nil {
			pt.Type_ = b.findType(pt.Name)
		}
	}
}

type TypeDefinition struct {
	Name string
	Type DataType
}

func (p *parser) parseTypeDefinition(b *Block) (*TypeDefinition, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	typeName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	dataType := p.parseType(b, true)

	return &TypeDefinition{Name: typeName, Type: dataType}, true
}

type DataType interface {
	Type() string // TODO: rename to TypeString
	Equals(dt DataType) bool
}

type RecordField struct {
	Identifier string
	Type       DataType
}

func (f *RecordField) String() string {
	var buf strings.Builder
	buf.WriteString(f.Identifier)
	buf.WriteString(" : ")
	buf.WriteString(f.Type.Type())
	return buf.String()
}

type RecordVariantField struct {
	TagField string
	Type     DataType
	TypeName string
	Variants []*RecordVariant
}

type RecordVariant struct {
	CaseLabels []ConstantLiteral
	Fields     *RecordType
}

func (p *parser) parseType(b *Block, resolvePointerTypesLater bool) DataType {
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
				return &PointerType{Name: ident}
			}
			p.errorf("unknown type %s", ident)
		}

		return &PointerType{Type_: typeDecl}
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
		return &SetType{ElementType: setDataType, Packed: packed}
	case itemFile:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after file, got %s", p.next())
		}
		p.next()
		fileDataType := p.parseType(b, resolvePointerTypesLater)
		return &FileType{ElementType: fileDataType, Packed: packed}
	case itemSign, itemUnsignedDigitSequence:
		// if the type definition is a sign or digits, it can only be a subrange type.
		return p.parseSubrangeType(b)
	default:
		p.errorf("unknown type %s", p.next().val)
	}
	// not reached.
	return nil
}

func (p *parser) parseEnumType(b *Block) *EnumType {
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

	return &EnumType{Identifiers: identifierList}
}

func (p *parser) parseIdentifierList(b *Block) []string {
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

type Variable struct {
	Name string
	Type DataType

	// the following fields are only set for variables that are looked up from within with statements,
	// and they indicate that Name and Type describe the field of a record variable of name BelongsTo of
	// type BelongsToType.
	BelongsTo     string
	BelongsToType DataType
}

func (p *parser) parseVarDeclarationPart(b *Block) {
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

func (p *parser) parseVariableDeclaration(b *Block) {
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
		if err := b.addVariable(&Variable{Name: varName, Type: dataType}); err != nil {
			p.errorf("%v", err)
		}
	}
}

func (p *parser) parseProcedureAndFunctionDeclarationPart(b *Block) {
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

type Routine struct {
	Name             string
	Block            *Block
	FormalParameters []*FormalParameter
	ReturnType       DataType
	Forward          bool // if true, routine is only forward-declared.
	varargs          bool // for builtin functions with variable arguments.
	isParameter      bool // if true, indicates that this refers to a procedural or functional parameter
}

func (p *parser) parseProcedureDeclaration(b *Block) {
	procedureName, parameterList := p.parseProcedureHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &Routine{Name: procedureName, FormalParameters: parameterList}

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

func (p *parser) parseProcedureHeading(b *Block) (string, []*FormalParameter) {
	if p.peek().typ != itemProcedure {
		p.errorf("expected procedure, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected procedure identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var parameterList []*FormalParameter
	if p.peek().typ == itemOpenParen {
		parameterList = p.parseFormalParameterList(b)
	}
	return procedureName, parameterList
}

type FormalParameter struct {
	Name              string
	Type              DataType
	VariableParameter bool
}

func (p *FormalParameter) String() string {
	var buf strings.Builder

	if p.VariableParameter {
		buf.WriteString("var ")
	}

	switch p.Type.(type) {
	case *FunctionType:
		buf.WriteString("function ")
		buf.WriteString(p.Name)
		buf.WriteString(p.Type.Type())
	case *ProcedureType:
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

func (p *parser) parseFormalParameterList(b *Block) []*FormalParameter {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	parameterList := []*FormalParameter{}

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

func (p *parser) parseFormalParameter(b *Block) []*FormalParameter {
	variableParam := false

	var formalParameters []*FormalParameter

	switch p.peek().typ {
	case itemProcedure:
		p.next()

		if p.peek().typ != itemIdentifier {
			p.errorf("expected procedure name, got %s instead", p.peek())
		}

		name := p.next().val

		params := p.parseFormalParameterList(b)

		formalParameters = append(formalParameters, &FormalParameter{
			Name: name,
			Type: &ProcedureType{FormalParams: params},
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

		formalParameters = append(formalParameters, &FormalParameter{
			Name: name,
			Type: &FunctionType{FormalParams: params, ReturnType: returnType},
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

		formalParameters = make([]*FormalParameter, 0, len(parameterNames))
		for _, name := range parameterNames {
			formalParameters = append(formalParameters, &FormalParameter{Name: name, Type: parameterType, VariableParameter: variableParam})
		}
	default:
		p.errorf("expected var, procedure, function or identifier, got %s", p.peek())
	}

	return formalParameters
}

func (p *parser) parseFunctionDeclaration(b *Block) {
	funcName, parameterList, returnType := p.parseFunctionHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	proc := &Routine{Name: funcName, FormalParameters: parameterList, ReturnType: returnType}

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

func (p *parser) parseFunctionHeading(b *Block) (string, []*FormalParameter, DataType) {
	if p.peek().typ != itemFunction {
		p.errorf("expected function, got %s", p.next())
	}
	p.next()

	if p.peek().typ != itemIdentifier {
		p.errorf("expected function identifier, got %s", p.next())
	}
	procedureName := p.next().val

	var (
		parameterList []*FormalParameter
		returnType    DataType
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

func (p *parser) parseStatementSequence(b *Block) []Statement {
	var statements []Statement

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

type StatementType int

const (
	StatementGoto StatementType = iota
	StatementAssignment
	StatementProcedureCall
	StatementCompoundStatement
	StatementWhile
	StatementRepeat
	StatementFor
	StatementIf
	StatementCase
	StatementWith
	StatementWrite
)

type Statement interface {
	Type() StatementType
	Label() *string
}

func (p *parser) parseStatement(b *Block) Statement {
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
		stmt = &LabelledStatement{label: label, Statement: stmt}
	}

	return stmt
}

func (p *parser) parseUnlabelledStatement(b *Block) Statement {
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
		return &GotoStatement{Target: tl}
	case itemIdentifier:
		return p.parseAssignmentOrProcedureStatement(b)
	case itemBegin:
		p.next()
		statements := p.parseStatementSequence(b)
		if p.peek().typ != itemEnd {
			p.errorf("expected end, got %s", p.next())
		}
		p.next()
		return &CompoundStatement{Statements: statements}
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

func (p *parser) parseAssignmentOrProcedureStatement(b *Block) Statement {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	identifier := p.next().val

	if p.peek().typ == itemOpenParen {
		if identifier == "writeln" {
			return p.parseWrite(b, true)
		} else if identifier == "write" {
			return p.parseWrite(b, false)
		}
		proc := b.findProcedure(identifier)
		if proc == nil {
			p.errorf("unknown procedure %s", identifier)
		}
		actualParameterList := p.parseActualParameterList(b)
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, actualParameterList); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &ProcedureCallStatement{Name: identifier, ActualParams: actualParameterList}
	}

	if identifier == "writeln" {
		return &WriteStatement{AppendNewLine: true}
	} else if identifier == "write" {
		p.errorf("write needs at least one parameter")
	}

	proc := b.findProcedure(identifier)
	if proc != nil {
		if err := p.validateParameters(proc.varargs, proc.FormalParameters, []Expression{}); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &ProcedureCallStatement{Name: identifier}
	}

	var lexpr Expression

	if funcDecl := b.findFunctionForAssignment(identifier); funcDecl != nil {
		lexpr = &VariableExpr{Name: identifier, Type_: funcDecl.ReturnType} // TODO: do we need a separate expression type for this?
	} else {
		lexpr = p.parseVariable(b, identifier)
	}

	if p.peek().typ == itemAssignment {
		p.next()
		if lexpr == nil {
			p.errorf("assignment: unknown left expression %s", identifier)
		}
		rexpr := p.parseExpression(b)
		if !lexpr.Type().Equals(rexpr.Type()) && !isCharStringLiteralAssignment(b, lexpr, rexpr) {
			p.errorf("incompatible types: got %s, expected %s", rexpr.Type().Type(), lexpr.Type().Type())
		}
		return &AssignmentStatement{LeftExpr: lexpr, RightExpr: rexpr}
	}

	p.errorf("unexpected token %s in statement", p.peek())
	// unreachable
	return nil
}

func (p *parser) parseWhileStatement(b *Block) *WhileStatement {
	if p.peek().typ != itemWhile {
		p.errorf("expected while, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)

	if !condition.Type().Equals(&BooleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &WhileStatement{Condition: condition, Statement: stmt}
}

func (p *parser) parseRepeatStatement(b *Block) *RepeatStatement {
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
	if !condition.Type().Equals(&BooleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	return &RepeatStatement{Condition: condition, Statements: stmts}
}

func (p *parser) parseForStatement(b *Block) *ForStatement {
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

	return &ForStatement{Name: variable, InitialExpr: initialExpr, FinalExpr: finalExpr, Statement: stmt, DownTo: down}
}

func (p *parser) parseIfStatement(b *Block) *IfStatement {
	if p.peek().typ != itemIf {
		p.errorf("expected if, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)
	if !condition.Type().Equals(&BooleanType{}) {
		p.errorf("condition is not boolean, but %s", condition.Type().Type())
	}

	if p.peek().typ != itemThen {
		p.errorf("expected then, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	var elseStmt Statement

	if p.peek().typ == itemElse {
		p.next()
		elseStmt = p.parseStatement(b)
	}

	return &IfStatement{Condition: condition, Statement: stmt, ElseStatement: elseStmt}
}

func (p *parser) parseCaseStatement(b *Block) Statement {
	if p.peek().typ != itemCase {
		p.errorf("expected case, got %s instead", p.peek())
	}
	p.next()

	expr := p.parseExpression(b)

	if p.peek().typ != itemOf {
		p.errorf("expected of, got %s instead", p.peek())
	}
	p.next()

	var caseLimbs []*CaseLimb

	limb := p.parseCaseLimb(b)
	for _, label := range limb.Label {
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
		for _, label := range limb.Label {
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

	return &CaseStatement{Expr: expr, CaseLimbs: caseLimbs}
}

func (p *parser) parseCaseLimb(b *Block) *CaseLimb {
	labels := p.parseCaseLabelList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead", p.peek())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &CaseLimb{
		Label:     labels,
		Statement: stmt,
	}
}

func (p *parser) parseWithStatement(b *Block) Statement {
	if p.peek().typ != itemWith {
		p.errorf("expected with, got %s instead", p.peek())
	}
	p.next()

	withBlock := &Block{
		Parent:  b,
		Routine: b.Routine,
	}

	var recordVariables []string

	for {

		if p.peek().typ != itemIdentifier {
			p.errorf("expected identifier of record variable, got %s instead", p.peek())
		}
		ident := p.next().val

		var typ DataType

		if varDecl := b.findVariable(ident); varDecl != nil {
			typ = varDecl.Type
		} else if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
			typ = paramDecl.Type
		} else {
			p.errorf("unknown variable %s x", ident)
		}

		recType, ok := typ.(*RecordType)
		if !ok {
			p.errorf("variable %s is not a record variable", ident)
		}

		recordVariables = append(recordVariables, ident)

		for _, field := range recType.Fields {
			withBlock.Variables = append(withBlock.Variables, &Variable{
				Name:          field.Identifier,
				Type:          field.Type,
				BelongsTo:     ident,
				BelongsToType: recType,
			})
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

	withBlock.Statements = append(withBlock.Statements, stmt)

	return &WithStatement{
		RecordVariables: recordVariables,
		Block:           withBlock,
	}
}

func (p *parser) parseActualParameterList(b *Block) []Expression {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	params := []Expression{}

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

func (p *parser) parseExpression(b *Block) Expression {
	p.logger.Printf("Parsing expression")

	expr := p.parseSimpleExpression(b)
	if !isRelationalOperator(p.peek().typ) {
		return expr.Reduce()
	}

	opToken := p.next()

	operator := itemTypeToRelationalOperator(opToken.typ)

	p.logger.Printf("Found relational operator %s after first simple expression", operator)

	rightExpr := p.parseSimpleExpression(b)

	p.logger.Printf("Finished parsing expression")

	relExpr := &RelationalExpr{
		Left:     expr,
		Operator: operator,
		Right:    rightExpr,
	}

	lt := relExpr.Left.Type()
	rt := relExpr.Right.Type()
	if operator == opIn {
		st, ok := rt.(*SetType)
		if !ok {
			p.errorf("in: expected set type, got %s instead.", rt.Type())
		}
		if !lt.Equals(st.ElementType) {
			p.errorf("type %s does not match set type %s", lt.Type(), st.ElementType.Type())
		}
	} else {
		ok := lt.Equals(rt)
		if !ok {
			p.errorf("can't %s %s %s", lt.Type(), relExpr.Operator, rt.Type())
		}
	}

	return relExpr.Reduce()
}

func (p *parser) parseSimpleExpression(b *Block) *SimpleExpr {
	p.logger.Printf("Parsing simple expression")
	var sign string
	if typ := p.peek().typ; typ == itemSign {
		sign = p.next().val
	}

	term := p.parseTerm(b)

	simpleExpr := &SimpleExpr{
		Sign:  sign,
		First: term,
	}

	if !isAdditionOperator(p.peek().typ) {
		p.logger.Printf("Finished parsing simple expression without further operators because %s is not an addition operator", p.peek())
		return simpleExpr
	}

	for {
		operatorToken := p.next()

		operator := tokenToAdditionOperator(operatorToken)

		if operator == OperatorOr {
			_, ok := simpleExpr.First.Type().(*BooleanType)
			if !ok {
				p.errorf("can't use or with %s", simpleExpr.First.Type().Type())
			}
		} else {
			if !isIntegerType(simpleExpr.First.Type()) && !isRealType(simpleExpr.First.Type()) {
				p.errorf("can only use %s operator with integer or real types, got %s instead", operator, simpleExpr.First.Type().Type())
			}
		}

		nextTerm := p.parseTerm(b)

		if !typesCompatible(simpleExpr.First.Type(), nextTerm.Type()) {
			p.errorf("can't %s %s %s", simpleExpr.First.Type().Type(), operator, nextTerm.Type().Type())
		}

		simpleExpr.Next = append(simpleExpr.Next, &Addition{Operator: operator, Term: nextTerm})

		if !isAdditionOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing simple expression because %s is not an addition operator", p.peek())

	return simpleExpr
}

func (p *parser) parseTerm(b *Block) *TermExpr {
	p.logger.Printf("Parsing term")
	factor := p.parseFactor(b)

	term := &TermExpr{
		First: factor,
	}

	if !isMultiplicationOperator(p.peek().typ) {
		p.logger.Printf("Finished parsing term without further operator because %s is not a multiplication operator", p.peek())
		return term
	}

	for {
		operator := itemTypeToMultiplicationOperator(p.next().typ)
		p.logger.Printf("parseTerm: got operator %s", operator)

		switch operator {
		case OperatorAnd:
			_, ok := term.First.Type().(*BooleanType)
			if !ok {
				p.errorf("can't use and with %s", term.First.Type().Type())
			}
		case OperatorMultiply:
			if !isIntegerType(term.First.Type()) && !isRealType(term.First.Type()) {
				p.errorf("can only use %s operator with integer or real types, got %s instead", operator, term.First.Type().Type())
			}
		case OperatorFloatDivide:
			if !isRealType(term.First.Type()) {
				p.errorf("can only use %s operator with real types, got %s instead", operator, term.First.Type().Type())
			}
		case OperatorDivide, OperatorModulo:
			if !isIntegerType(term.First.Type()) {
				p.errorf("can only use %s operator with integer types, got %s intead", operator, term.First.Type().Type())
			}
		}

		nextFactor := p.parseFactor(b)

		if !typesCompatible(term.First.Type(), nextFactor.Type()) {
			p.errorf("can't %s %s %s", term.First.Type().Type(), operator, nextFactor.Type().Type())
		}

		term.Next = append(term.Next, &Multiplication{Operator: operator, Factor: nextFactor})

		if !isMultiplicationOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing term because %s is not a multiplication operator", p.peek())

	return term
}

func (p *parser) parseFactor(b *Block) Expression {
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
				return &FunctionCallExpr{Name: ident, ActualParams: params, Type_: funcDecl.ReturnType}
			}

			if len(funcDecl.FormalParameters) > 0 { // function has formal parameter which are not provided -> it's a functional-parameter
				return &VariableExpr{Name: ident, Type_: &FunctionType{FormalParams: funcDecl.FormalParameters, ReturnType: funcDecl.ReturnType}}
			}
			// TODO: what if function has no formal parameters? is it a functional parameter or a function call? needs resolved later, probably.
			if err := p.validateParameters(funcDecl.varargs, funcDecl.FormalParameters, []Expression{}); err != nil {
				p.errorf("function %s: %v", ident, err)
			}
			return &FunctionCallExpr{Name: ident, Type_: funcDecl.ReturnType}

		}
		if constDecl := b.findConstantDeclaration(ident); constDecl != nil {
			return &ConstantExpr{ident, constDecl.Value.ConstantType()}
		}
		if idx, typ := b.findEnumValue(ident); typ != nil {
			return &EnumValueExpr{Name: ident, Value: idx, Type_: typ}
		}

		return p.parseVariable(b, ident)
	case itemSign:
		sign := p.next().val
		return p.parseNumber(sign == "-")
	case itemUnsignedDigitSequence:
		return p.parseNumber(false)
	case itemStringLiteral:
		p.logger.Printf("parseFactor: got string literal %s", p.peek())
		return &StringExpr{p.next().val}
	case itemOpenBracket:
		return p.parseSet(b)
	case itemNil:
		p.next()
		return &NilExpr{}
	case itemOpenParen:
		return p.parseSubExpr(b)
	case itemNot:
		p.next()
		expr := p.parseFactor(b)
		if !expr.Type().Equals(&BooleanType{}) {
			p.errorf("can't NOT %s", expr.Type().Type())
		}
		return &NotExpr{expr}
	default:
		p.errorf("unexpected %s while parsing factor", p.peek())
	}
	// unreachable
	return nil
}

func (p *parser) parseVariable(b *Block, ident string) Expression {
	var expr Expression

	if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
		expr = &VariableExpr{Name: ident, Type_: paramDecl.Type} // TODO: do we need a separate formal parameter expression here?
	} else if procDecl := b.findProcedure(ident); procDecl != nil { // TODO: do we need a separate procedural parameter expression here?
		expr = &VariableExpr{Name: ident, Type_: &ProcedureType{FormalParams: procDecl.FormalParameters}}
	} else if varDecl := b.findVariable(ident); varDecl != nil {
		expr = &VariableExpr{Name: ident, Type_: varDecl.Type}
	}

	if expr == nil {
		p.errorf("unknown identifier %s", ident)
	}

	cont := true

	for cont {
		switch p.peek().typ {
		case itemCaret:
			_, ok := expr.Type().(*PointerType)
			if !ok {
				p.errorf("attempting to ^ but expression is not a pointer type")
			}
			p.next()
			expr = &DerefExpr{Expr: expr}
		case itemOpenBracket:
			expr = p.parseIndexVariableExpr(b, expr)
		case itemDot:
			p.next()

			rt, ok := expr.Type().(*RecordType)
			if !ok {
				p.errorf("expression is a record type")
			}

			if p.peek().typ != itemIdentifier {
				p.errorf("expected identifier, got %s instead", p.peek())
			}
			fieldIdentifier := p.next().val
			field := rt.findField(fieldIdentifier)

			expr = &FieldDesignatorExpr{Expr: expr, Field: fieldIdentifier, Type_: field.Type}
		default:
			cont = false
		}
	}

	return expr
}

func (p *parser) parseNumber(minus bool) Expression {
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
		return &RealExpr{Minus: minus, BeforeComma: unsignedDigitSequence, AfterComma: afterComma, ScaleFactor: scaleFactor}
	}
	intValue, err := strconv.ParseInt(unsignedDigitSequence, 10, 64)
	if err != nil {
		p.errorf("failed to parse %s as integer: %v", unsignedDigitSequence, err)
	}
	if minus {
		intValue = -intValue
	}
	p.logger.Printf("parseNumber: parsed int %d", intValue)
	return &IntegerExpr{intValue}
}

func (p *parser) parseScaleFactor() int {
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

func (p *parser) parseSet(b *Block) *SetExpr {
	if p.peek().typ != itemOpenBracket {
		p.errorf("expected [, found %s instead", p.next())
	}
	p.next()

	set := &SetExpr{}

	expr := p.parseExpression(b)
	set.Elements = append(set.Elements, expr)

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
		set.Elements = append(set.Elements, expr)
	}

	return set
}

func (p *parser) parseSubExpr(b *Block) *SubExpr {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s instead", p.peek())
	}
	p.next()

	expr := p.parseExpression(b)

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s instead", p.peek())
	}
	p.next()

	return &SubExpr{expr}
}

func (p *parser) parseExpressionList(b *Block) []Expression {
	var exprs []Expression

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

func (p *parser) parseArrayType(b *Block, packed bool) *ArrayType {
	if p.peek().typ != itemArray {
		p.errorf("expected array, got %s instead", p.peek())
	}
	p.next()

	if p.peek().typ != itemOpenBracket {
		p.errorf("expected [, got %s instead", p.peek())
	}
	p.next()

	indexTypes := []DataType{p.parseSimpleType(b)}

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

	return &ArrayType{
		IndexTypes:  indexTypes,
		ElementType: elementType,
		Packed:      packed,
	}
}

func (p *parser) parseSimpleType(b *Block) DataType {
	if p.peek().typ == itemOpenParen {
		return p.parseEnumType(b)
	}

	return p.parseSubrangeType(b)
}

func (p *parser) parseSubrangeType(b *Block) DataType {
	lowerBound := p.parseConstant(b)
	var (
		lowerValue int
		upperValue int
		typ        DataType
		upperType  DataType
	)

	switch lb := lowerBound.(type) {
	case *IntegerLiteral:
		lowerValue = lb.Value
		typ = lb.ConstantType()
	case *EnumValueLiteral:
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
	case *IntegerLiteral:
		upperValue = ub.Value
		upperType = ub.ConstantType()
	case *EnumValueLiteral:
		upperValue = ub.Value
		upperType = ub.Type
	default:
		p.errorf("expected upper bound to be an integer or an enum value, got a %s instead", ub.ConstantType().Type())
	}

	if !upperType.Equals(typ) {
		p.errorf("type of lower bound differs from upper bound: %s vs %s", typ.Type(), upperType.Type())
	}

	return &SubrangeType{
		LowerBound: lowerValue,
		UpperBound: upperValue,
		Type_:      typ,
	}
}

func (p *parser) parseConstant(b *Block) ConstantLiteral {
	minus := false
	if p.peek().typ == itemSign {
		minus = p.next().val == "-"
	}
	return p.parseConstantWithoutSign(b, minus)
}

func isPossiblyConstant(b *Block, it item) bool {
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

func (p *parser) parseConstantWithoutSign(b *Block, minus bool) ConstantLiteral {
	var v ConstantLiteral

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

			v = &EnumValueLiteral{Symbol: constantName, Value: idx, Type: typ}
		}
	} else if p.peek().typ == itemUnsignedDigitSequence {
		number := p.parseNumber(false) // negation will be done later on.
		switch n := number.(type) {
		case *IntegerExpr:
			v = &IntegerLiteral{Value: int(n.Value)}
		case *RealExpr:
			v = &RealLiteral{Minus: n.Minus, BeforeComma: n.BeforeComma, AfterComma: n.AfterComma, ScaleFactor: n.ScaleFactor}
		}
	} else if p.peek().typ == itemStringLiteral {
		v = &StringLiteral{Value: p.next().val}
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

func (p *parser) parseRecordType(b *Block, packed bool) *RecordType {
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

func (p *parser) parseFieldList(b *Block, packed bool) *RecordType {
	record := &RecordType{Packed: packed}

	if p.peek().typ != itemCase && p.peek().typ != itemIdentifier {
		// if it's neither a case nor an identifier, we probably have an empty field list
		return record
	}

	if p.peek().typ != itemCase {
		record.Fields = p.parsedFixedPart(b)
	}

	if p.peek().typ == itemCase {
		record.VariantField = p.parseVariantField(b, packed)
	}

	if p.peek().typ == itemSemicolon {
		p.next()
	}

	fieldNames := map[string]bool{}

	for _, f := range record.Fields {
		if fieldNames[f.Identifier] {
			p.errorf("duplicate field name %s", f.Identifier)
		}
		fieldNames[f.Identifier] = true
	}

	if record.VariantField != nil {
		for _, v := range record.VariantField.Variants {
			for _, f := range v.Fields.Fields {
				if fieldNames[f.Identifier] {
					p.errorf("duplicate variant field name %s", f.Identifier)
				}
				fieldNames[f.Identifier] = true
			}
		}
	}

	return record
}

func (p *parser) parsedFixedPart(b *Block) (fields []*RecordField) {
	sectionFields := p.parseRecordSection(b)
	fields = append(fields, sectionFields...)

	for {
		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()

		if p.peek().typ != itemIdentifier {
			break
		}

		sectionFields := p.parseRecordSection(b)
		fields = append(fields, sectionFields...)
	}

	return fields
}

func (p *parser) parseVariantField(b *Block, packed bool) (field *RecordVariantField) {
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

	field = &RecordVariantField{
		TagField: tag,
		Type:     typeDef,
		TypeName: typeIdentifier,
	}

	for {
		if !isPossiblyConstant(b, p.peek()) { // variant always starts with constant.
			break
		}

		variant := p.parseVariant(b, packed)
		field.Variants = append(field.Variants, variant)

		if p.peek().typ != itemSemicolon {
			break
		}
		p.next()
	}

	return field
}

func (p *parser) parseVariant(b *Block, packed bool) *RecordVariant {
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

	return &RecordVariant{
		CaseLabels: labels,
		Fields:     fieldList,
	}
}

func (p *parser) parseCaseLabelList(b *Block) (labels []ConstantLiteral) {
	label := p.parseConstant(b)
	labels = append(labels, label)

	for p.peek().typ == itemComma {
		p.next()

		label := p.parseConstant(b)
		labels = append(labels, label)
	}

	return labels
}

func (p *parser) parseRecordSection(b *Block) []*RecordField {
	identifierList := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead.", p.peek())
	}
	p.next()

	typ := p.parseType(b, false)

	var fields []*RecordField

	for _, ident := range identifierList {
		fields = append(fields, &RecordField{Identifier: ident, Type: typ})
	}
	return fields
}

func (p *parser) validateParameters(varargs bool, formalParams []*FormalParameter, actualParams []Expression) error {
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

func (p *parser) parseIndexVariableExpr(b *Block, expr Expression) *IndexedVariableExpr {
	p.next()
	indexes := p.parseExpressionList(b)
	if p.peek().typ != itemCloseBracket {
		p.errorf("expected ], got %s instead", p.peek())
	}
	p.next()

	_, isStringType := expr.Type().(*StringType)
	if isStringType {
		if len(indexes) != 1 {
			p.errorf("strings have exactly 1 dimension, actually got %d", len(indexes))
		}

		if !isIntegerType(indexes[0].Type()) {
			p.errorf("string index needs to be an integer type, actually got %s", indexes[0].Type().Type())
		}

		return &IndexedVariableExpr{Expr: expr, IndexExprs: indexes, Type_: &CharType{}}
	}

	arrType, ok := expr.Type().(*ArrayType)
	if !ok {
		p.errorf("expression is not array")
	}

	// TODO: support situation where fewer index expressions mean that an array of fewer dimensions is returned.

	if len(arrType.IndexTypes) != len(indexes) {
		p.errorf("array has %d dimensions but %d index expressions were provided", len(arrType.IndexTypes), len(indexes))
	}

	for idx, idxType := range arrType.IndexTypes {
		if !typesCompatible(idxType, indexes[idx].Type()) {
			p.errorf("array dimension %d is of type %s, but index expression type %s was provided", idx, idxType.Type(), indexes[idx].Type().Type())
		}
	}

	return &IndexedVariableExpr{Expr: expr, IndexExprs: indexes, Type_: arrType.ElementType}
}

func (p *parser) parseWrite(b *Block, ln bool) *WriteStatement {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s instead", p.peek())
	}
	p.next()

	stmt := &WriteStatement{AppendNewLine: ln}

	first := p.parseExpression(b)

	_, isFileType := first.Type().(*FileType)

	if first.IsVariableExpr() && isFileType {
		stmt.FileVar = first
	} else {
		p.verifyWriteType(first.Type(), ln)
		width, decimalPlaces := p.parseWritelnFormat(first, b)
		stmt.ActualParams = append(stmt.ActualParams, &FormatExpr{Expr: first, Width: width, DecimalPlaces: decimalPlaces})
	}

	for p.peek().typ == itemComma {
		p.next()

		param := p.parseExpression(b)
		p.verifyWriteType(param.Type(), ln)

		width, decimalPlaces := p.parseWritelnFormat(param, b)
		stmt.ActualParams = append(stmt.ActualParams, &FormatExpr{Expr: param, Width: width, DecimalPlaces: decimalPlaces})
	}

	if p.peek().typ != itemCloseParen {
		p.errorf("expected ), got %s instead", p.peek())
	}
	p.next()

	return stmt
}

func (p *parser) parseWritelnFormat(expr Expression, b *Block) (widthExpr Expression, decimalPlacesExpr Expression) {
	if p.peek().typ == itemColon {
		p.next()
		widthExpr = p.parseExpression(b)
		if p.peek().typ == itemColon {
			p.next()
			decimalPlacesExpr = p.parseExpression(b)
		}
	}

	if decimalPlacesExpr != nil && !expr.Type().Equals(&RealType{}) {
		p.errorf("decimal places format is not allowed for type %s", expr.Type().Type())
	}

	return widthExpr, decimalPlacesExpr
}

func (p *parser) verifyWriteType(typ DataType, ln bool) {
	funcName := "write"
	if ln {
		funcName += "ln"
	}

	allowedWriteTypes := []DataType{&IntegerType{}, &RealType{}, &CharType{}, getBuiltinType("string")}

	for _, at := range allowedWriteTypes {
		if at.Equals(typ) {
			return
		}
	}

	p.errorf("can't use variables of type %s with %s", typ.Type(), funcName)
}
