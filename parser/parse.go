package parser

import (
	"errors"
	"fmt"
	"io"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
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

// AST describes the Abstract Syntax Tree of the parsed Pascal program.
type AST struct {
	// Program name
	Name string

	// Files provided in the program heading.
	Files []string

	// Block contains the top-most block of the program that contains all global
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

// parse parses a Pascal program.
//
//	program =
//	    program-heading block "." .
func (p *parser) parse() (ast *AST, err error) {
	defer p.recover(&err)

	ast = &AST{}

	p.parseProgramHeading(ast)

	ast.Block = p.parseBlock(builtinBlock, nil)

	if p.peek().typ != itemDot {
		p.errorf("expected ., got %s instead", p.next())
	}
	p.next()

	return ast, nil
}

// parseProgramHeading parses a program heading.
//
//	program-heading =
//	    "program" identifier [ "(" identifier-list ")" ] ";".
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

// parseBlock parses a block.
//
//	block =
//	    declaration-part statement-part .
func (p *parser) parseBlock(parent *Block, proc *Routine) *Block {
	b := &Block{
		Parent:  parent,
		Routine: proc,
	}
	p.parseDeclarationPart(b)
	p.parseStatementPart(b)
	return b
}

// parseDeclaration parses a declaration part.
//
//	declaration-part
//	    [ label-declaration-part ]
//	    [ constant-definition-part ]
//	    [ type-definition-part ]
//	    [ variable-declaration-part ]
//	    procedure-and-function-declaration-part
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

// parseStatementPart parses a statement part.
//
//	statement-part =
//	    "begin" [ statement-sequence ] "end"
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

// parseLabelDeclarationPart parses a label declaration part.
//
//	label-declaration-part =
//	    "label" label { "," label } ";" .
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

// parseConstantDefinitionPart parses a constant definition part.
//
//	constant-definition-part =
//	    "const" constant-definition ";" { constant-definition ";" } .
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

// parseConstantDefinition parses a constant definition.
//
//	constant-definition =
//	    identifier "=" constant .
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

// parseTypeDefinitionPart parses a type definition part.
//
//	type-definition-part =
//	    "type" type-definition ";" { type-definition ";" } .
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
		if err := typeDef.Type.Resolve(b); err != nil {
			p.errorf("couldn't resolve type: %v", err)
		}
	}
}

type TypeDefinition struct {
	Name string
	Type DataType
}

// parseTypeDefinition parses a type definition.
//
//	type-definition =
//	   identifier "=" type .
func (p *parser) parseTypeDefinition(b *Block) (*TypeDefinition, bool) {
	if p.peek().typ != itemIdentifier {
		return nil, false
	}

	typeName := p.next().val

	if p.peek().typ != itemEqual {
		p.errorf("expected =, got %s", p.next())
	}
	p.next()

	dataType := p.parseType(b)

	return &TypeDefinition{Name: typeName, Type: dataType}, true
}

type RecordField struct {
	Identifier string
	Type       DataType
}

func (f *RecordField) String() string {
	var buf strings.Builder
	buf.WriteString(f.Identifier)
	buf.WriteString(" : ")
	buf.WriteString(f.Type.TypeString())
	return buf.String()
}

type RecordVariantField struct {
	TagField string
	Type     DataType
	Variants []*RecordVariant
}

type RecordVariant struct {
	CaseLabels []ConstantLiteral
	Fields     *RecordType
}

// parseType parses a type. While the EBNF looks neat, the reality is little bit messier.
//
//	type =
//	    simple-type | structured-type | pointer-type | type-identifier .
//	simple-type =
//	    subrange-type | enumerated-type
//	structured-type =
//	    [ "packed" ] unpacked-structured-type
//	unpacked-structured-type =
//	    array-type | record-type | set-type | file-type .
//	set-type =
//	    "set" "of" base-type .
//	base-type =
//	     type .
//	file-type =
//	    "file" "of" file-component-type .
//	file-component-type =
//	    type .
//	pointer-type =
//	    "^" type-identifier .
//	type-identifier =
//	    identifier .
func (p *parser) parseType(b *Block) DataType {
	packed := false

restartParseDataType:
	switch p.peek().typ {
	case itemIdentifier:
		ident := p.peek().val
		// if identifier is an already existing type name, it's an alias.
		if typ := b.findType(ident); typ != nil {
			p.next()
			return typ.Named(ident)
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

		if typ := getBuiltinType(ident); typ != nil {
			return &PointerType{Name: ident, Type_: typ}
		}

		return &PointerType{Name: ident, block: b}

		/*

			typeDecl := b.findType(ident)
			if typeDecl == nil {
				fmt.Printf("pointer type that needs resolving later: %s\n", ident)
				if resolvePointerTypesLater {
					return &PointerType{Name: ident}
				}
				p.errorf("unknown type %s", ident)
			}

			return &PointerType{Name: ident}

			// don't store names of built-in types, as they are fully represented in the type declaration already, and
			// the name would only stand in the way for any code generation that assumes that non-empty names refer
			// to non-builtin types.
			if getBuiltinType(ident) != nil {
				ident = ""
			}

			fmt.Printf("pointer type: %s : %s\n", ident, typeDecl.Type())

			return &PointerType{Name: ident, Type_: typeDecl.Named(ident)}
		*/
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
		setDataType := p.parseType(b)
		if !isOrdinalType(setDataType) {
			p.errorf("sets require an ordinal type, got %s instead", setDataType.TypeString())
		}
		return &SetType{ElementType: setDataType, Packed: packed}
	case itemFile:
		p.next()
		if p.peek().typ != itemOf {
			p.errorf("expected of after file, got %s", p.next())
		}
		p.next()
		fileDataType := p.parseType(b)
		return &FileType{ElementType: fileDataType, Packed: packed}
	case itemSign, itemUnsignedDigitSequence, itemStringLiteral:
		// if the type definition is a sign, digits or a string (really char) literal, it can only be a subrange type.
		return p.parseSubrangeType(b)
	default:
		p.errorf("unknown type %s", p.next().val)
	}
	// not reached.
	return nil
}

// parseEnumType parses an enumerated type.
//
//	enumerated-type =
//	   "(" identifier-list ")" .
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

// parseIdentifierList parses an identifier list.
//
//	identifier-list =
//	    identifier { "," identifier } .
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
	IsRecordField bool // true if "variable" is a record field.
	BelongsToExpr Expression
	BelongsToType DataType
}

// parseVarDeclarationPart parses a variable declaration part.
//
//	variable-declaration-part =
//		"var" variable-declaration ";" { variable-declaration ";" } .
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

// parseVariableDeclaration parses a variable declaration.
//
//	variable-declaration =
//		identifier-list ":" type .
func (p *parser) parseVariableDeclaration(b *Block) {
	variableNames := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s", p.next())
	}
	p.next()

	dataType := p.parseType(b)

	if err := dataType.Resolve(b); err != nil {
		p.errorf("variables %s: %s", strings.Join(variableNames, ", "), err)
	}

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

// parseProcedureAndFunctionDeclarationPart parses a procedure and function declaration part.
//
//	procedure-and-function-declaration-part =
//		{ (procedure-declaration | function-declaration) ";" } .
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
	isParameter      bool // if true, indicates that this refers to a procedural or functional parameter
	validator        func([]Expression) (DataType, error)
}

// parseProcedureDeclaration parses a procedure declaration.
//
//	procedure-declaration =
//		procedure-heading ";" procedure-body |
//		procedure-heading ";" directive |
//		procedure-identification ";" procedure-body .
func (p *parser) parseProcedureDeclaration(b *Block) {
	procedureName, parameterList := p.parseProcedureHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	forwardDeclProc := &Routine{Name: procedureName, FormalParameters: parameterList, Forward: true}
	proc := &Routine{Name: procedureName, FormalParameters: parameterList}

	if p.peek().typ == itemForward {
		p.next()
		if err := b.addProcedure(forwardDeclProc); err != nil {
			p.errorf("%v", err)
		}
		return
	}

	if procDecl := b.findProcedure(procedureName); procDecl == nil {
		if err := b.addProcedure(forwardDeclProc); err != nil {
			p.errorf("%v", err)
		}
	} else if proc.FormalParameters == nil {
		proc.FormalParameters = procDecl.FormalParameters
	}

	for _, param := range proc.FormalParameters {
		if err := param.Type.Resolve(b); err != nil {
			p.errorf("parameter %s: %v", param.Name, err)
		}
	}

	proc.Block = p.parseBlock(b, proc)

	if err := b.addProcedure(proc); err != nil {
		p.errorf("%v", err)
	}
}

// parseProcedureHeading parses a procedure heading.
//
//	procedure-heading =
//		"procedure" identifier [ formal-parameter-list ] .
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
		buf.WriteString(p.Type.TypeString())
	case *ProcedureType:
		buf.WriteString("procedure ")
		buf.WriteString(p.Name)
		buf.WriteString(p.Type.TypeString())
	default:
		buf.WriteString(p.Name)
		buf.WriteString(" : ")
		buf.WriteString(p.Type.TypeString())
	}

	return buf.String()
}

// parseFormalParameterList parses a formal parameter list.
//
//	formal-parameter-list =
//		"(" formal-parameter-section { ";" formal-parameter-section } ")" .
func (p *parser) parseFormalParameterList(b *Block) []*FormalParameter {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s", p.next())
	}
	p.next()

	parameterList := []*FormalParameter{}

parameterListLoop:
	for {

		formalParameters := p.parseFormalParameterSection(b)

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

// parseFormalParameterSection parses a formal parameter section.
//
//	formal-parameter-section =
//		value-parameter-section |
//		variable-parameter-section |
//		procedure-parameter-section |
//		function-parameter-section .
//	value-parameter-section =
//		identifier-list ":" parameter-type .
//	variable-parameter-section =
//		var identifier-list ":" parameter-type .
//	procedure-parameter-section =
//		procedure-heading .
//	function-parameter-section =
//		function-heading .
func (p *parser) parseFormalParameterSection(b *Block) []*FormalParameter {
	variableParam := false

	var formalParameters []*FormalParameter

	switch p.peek().typ {
	case itemProcedure:
		p.next()

		if p.peek().typ != itemIdentifier {
			p.errorf("expected procedure name, got %s instead", p.peek())
		}

		name := p.next().val

		var params []*FormalParameter

		if p.peek().typ == itemOpenParen {
			params = p.parseFormalParameterList(b)
		}

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

		var params []*FormalParameter

		if p.peek().typ == itemOpenParen {
			params = p.parseFormalParameterList(b)
		}

		if p.peek().typ != itemColon {
			p.errorf("expected : after formal parameter list, got %s instead", p.peek())
		}

		p.next()

		returnType := p.parseType(b)

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

		parameterType := p.parseType(b)

		formalParameters = make([]*FormalParameter, 0, len(parameterNames))
		for _, name := range parameterNames {
			formalParameters = append(formalParameters, &FormalParameter{Name: name, Type: parameterType, VariableParameter: variableParam})
		}
	default:
		p.errorf("expected var, procedure, function or identifier, got %s", p.peek())
	}

	return formalParameters
}

// parseFunctionDeclaration parses a function declaration.
//
//	function-declaration =
//		function-heading ";" function-body |
//		function-heading ";" directive |
//		function-identification ";" function-body .
func (p *parser) parseFunctionDeclaration(b *Block) {
	funcName, parameterList, returnType := p.parseFunctionHeading(b)

	if p.peek().typ != itemSemicolon {
		p.errorf("expected ;, got %s", p.next())
	}
	p.next()

	forwardDeclProc := &Routine{Name: funcName, FormalParameters: parameterList, ReturnType: returnType, Forward: true}
	proc := &Routine{Name: funcName, FormalParameters: parameterList, ReturnType: returnType}

	// if it is a true forward declaration, we just add the forward function and then return.
	if p.peek().typ == itemForward {
		p.next()
		if err := b.addFunction(forwardDeclProc); err != nil {
			p.errorf("%v", err)
		}
		return
	}
	// right after parsing the function heading, we add even non-forward function declarations
	// as forward declarations if there isn't one already, so that when parsing the function block, the function that
	// is currently being declared is already available. This is necessary for correctly
	// parsing recursive functions.
	if funcDecl := b.findFunction(funcName); funcDecl == nil {
		if err := b.addFunction(forwardDeclProc); err != nil {
			p.errorf("%v", err)
		}
	} else if proc.FormalParameters == nil {
		// this is necessary because the formal parameters and return type of the proper declaration may not be present
		// but are present in the forward declaration.
		proc.FormalParameters = funcDecl.FormalParameters
		proc.ReturnType = funcDecl.ReturnType
	}

	for _, param := range proc.FormalParameters {
		if err := param.Type.Resolve(b); err != nil {
			p.errorf("parameter %s: %v", param.Name, err)
		}
	}

	if err := proc.ReturnType.Resolve(b); err != nil {
		p.errorf("return type: %v", err)
	}

	proc.Block = p.parseBlock(b, proc)

	if err := b.addFunction(proc); err != nil {
		p.errorf("%v", err)
	}
}

// parseFunctionHeading parses a function heading.
//
//	function-heading =
//		"function" identifier [ formal-parameter-list ] ":" result-type .
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
		returnType = p.parseType(b)
	}

	return procedureName, parameterList, returnType
}

// parseStatementSequence parses a statement sequence.
//
//	statement-sequence =
//		statement { ";" statement } .
func (p *parser) parseStatementSequence(b *Block) []Statement {
	var statements []Statement

	// unlike EBNF, allow empty statement sequence.
	if p.peek().typ == itemEnd || p.peek().typ == itemUntil {
		return statements
	}

	statements = append(statements, p.parseStatement(b))

	for {
		// if there is no semicolon after the statement, end the loop
		if p.peek().typ != itemSemicolon {
			break
		}

		// consume any empty statement semicolons.
		for p.peek().typ == itemSemicolon {
			p.next()
		}

		// if we now find end or until, then we have also found the end of the statement sequence.
		if p.peek().typ == itemEnd || p.peek().typ == itemUntil {
			break
		}

		// now we should have a statement for sure, which we will parse.
		statements = append(statements, p.parseStatement(b))
	}

	return statements
}

// parseStatement parses a statement.
//
//	statement =
//		[ label ":" ] (simple-statement | structured-statement) .
func (p *parser) parseStatement(b *Block) Statement {
	var label *string
	if p.peek().typ == itemUnsignedDigitSequence {
		labelStr := p.next().val
		label = &labelStr

		if !b.isValidLabel(labelStr) {
			p.errorf("undeclared label %s", labelStr)
		}

		if p.peek().typ != itemColon {
			p.errorf("expected : after label, got %s", p.next())
		}
		p.next()
	}

	return p.parseUnlabelledStatement(b, label)
}

// parseUnlabelledStatement parses a statement without the prepended label.
//
//	simple-statement =
//		[ assignment-statement | procedure-statement | goto-statement ] .
//	assignment-statement =
//		(variable | function-identifier) ":=" expression .
//	procedure-statement =
//		procedure-identifier [ actual-parameter-list ] .
//	goto-statement =
//		"goto" label .
func (p *parser) parseUnlabelledStatement(b *Block, label *string) Statement {
restart:
	switch p.peek().typ {
	case itemSemicolon: // ignore semicolons before statements.
		p.next()
		goto restart
	case itemGoto:
		p.next()
		if p.peek().typ != itemUnsignedDigitSequence {
			p.errorf("expected label after goto, got %s", p.next())
		}
		tl := p.next().val
		i, err := strconv.ParseInt(tl, 10, 64)
		if err != nil {
			p.errorf("invalid label %q: %v", tl, err)
		}
		tl = fmt.Sprint(i)
		if !b.isValidLabel(tl) {
			p.errorf("invalid goto label %s", tl)
		}
		return &GotoStatement{label: label, Target: tl}
	case itemIdentifier:
		return p.parseAssignmentOrProcedureStatement(b, label)
	case itemBegin:
		p.next()
		statements := p.parseStatementSequence(b)
		if p.peek().typ != itemEnd {
			p.errorf("expected end, got %s", p.next())
		}
		p.next()
		return &CompoundStatement{label: label, Statements: statements}
	case itemWhile:
		return p.parseWhileStatement(b, label)
	case itemRepeat:
		return p.parseRepeatStatement(b, label)
	case itemFor:
		return p.parseForStatement(b, label)
	case itemIf:
		return p.parseIfStatement(b, label)
	case itemCase:
		return p.parseCaseStatement(b, label)
	case itemWith:
		return p.parseWithStatement(b, label)
	}
	p.errorf("unsupported %s as statement", p.next())
	return nil
}

// parseAssignmentOrProcedureStatement parses an assignment or a procedure statement. Please
// note that the structure of the program doesn't strictly match the structure of the EBNF.
func (p *parser) parseAssignmentOrProcedureStatement(b *Block, label *string) Statement {
	if p.peek().typ != itemIdentifier {
		p.errorf("expected identifier, got %s", p.next())
	}

	identifier := p.next().val

	if p.peek().typ == itemOpenParen {
		if identifier == "writeln" {
			return p.parseWrite(b, true, label)
		} else if identifier == "write" {
			return p.parseWrite(b, false, label)
		}
		proc := b.findProcedure(identifier)
		if proc == nil {
			p.errorf("unknown procedure %s", identifier)
		}
		actualParameterList := p.parseActualParameterList(b)
		if _, err := p.validateParameters(proc, actualParameterList); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &ProcedureCallStatement{label: label, Name: identifier, ActualParams: actualParameterList, FormalParams: proc.FormalParameters}
	}

	if identifier == "writeln" {
		return &WriteStatement{label: label, AppendNewLine: true}
	} else if identifier == "write" {
		p.errorf("write needs at least one parameter")
	}

	proc := b.findProcedure(identifier)
	if proc != nil {
		if _, err := p.validateParameters(proc, []Expression{}); err != nil {
			p.errorf("procedure %s: %v", identifier, err)
		}
		return &ProcedureCallStatement{label: label, Name: identifier, FormalParams: proc.FormalParameters}
	}

	var lexpr Expression

	if funcDecl := b.findFunctionForAssignment(identifier); funcDecl != nil {
		lexpr = &VariableExpr{Name: identifier, Type_: funcDecl.ReturnType, IsReturnValue: true}
	} else {
		lexpr = p.parseVariable(b, identifier)
	}

	if p.peek().typ == itemAssignment {
		p.next()
		if lexpr == nil {
			p.errorf("assignment: unknown left expression %s", identifier)
		}
		rexpr := p.parseExpression(b)
		if lexpr.Type() == nil {
			p.errorf("in assignment, left expression type is nil")
		}
		if rexpr.Type() == nil {
			p.errorf("in assignment, right expression type is nil")
		}
		if !typesCompatibleForAssignment(lexpr.Type(), rexpr.Type()) {
			if !isCharStringLiteralAssignment(b, lexpr, rexpr) {
				p.errorf("incompatible types: got %s, expected %s", rexpr.Type().TypeString(), lexpr.Type().TypeString())
			} else {
				rexpr = p.stringToCharLiteralExpr(rexpr)
			}
		}
		return &AssignmentStatement{label: label, LeftExpr: lexpr, RightExpr: rexpr}
	}

	p.errorf("unexpected token %s in statement", p.peek())
	// unreachable
	return nil
}

func (p *parser) stringToCharLiteralExpr(expr Expression) Expression {
	if se, ok := expr.(*StringExpr); ok {
		return &CharExpr{
			Value: se.Value[0],
		}
	}

	if ce, ok := expr.(*ConstantExpr); ok {
		return ce
	}

	p.errorf("expected string literal or constant expression, got %T", expr)
	return nil
}

// parseWhileStatement parses a while statement.
//
//	while-statement =
//		"while" expression "do" statement .
func (p *parser) parseWhileStatement(b *Block, label *string) *WhileStatement {
	if p.peek().typ != itemWhile {
		p.errorf("expected while, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)

	if !IsBooleanType(condition.Type()) {
		p.errorf("condition is not boolean, but %s", condition.Type().TypeString())
	}

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s", p.next())
	}
	p.next()

	stmt := p.parseStatement(b)

	return &WhileStatement{label: label, Condition: condition, Statement: stmt}
}

// parseRepeatStatement parses a repeat statement.
//
//	repeat-statement =
//		"repeat" statement-sequence "until" expression .
func (p *parser) parseRepeatStatement(b *Block, label *string) *RepeatStatement {
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
	if !IsBooleanType(condition.Type()) {
		p.errorf("condition is not boolean, but %s", condition.Type().TypeString())
	}

	return &RepeatStatement{label: label, Condition: condition, Statements: stmts}
}

// parseForStatement parses a for statement.
//
//	for-statement =
//		"for" variable-identifier ":=" initial-expression ("to" | "downto") final-expression "do" statement .
//	initial-expression =
//		expression .
//	final-expression =
//		expression .
func (p *parser) parseForStatement(b *Block, label *string) *ForStatement {
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

	return &ForStatement{label: label, Name: variable, InitialExpr: initialExpr, FinalExpr: finalExpr, Statement: stmt, DownTo: down}
}

// parseIfStatement parses an if statement.
//
//	if-statement =
//		"if" expression "then" statement [ "else" statement ] .
func (p *parser) parseIfStatement(b *Block, label *string) *IfStatement {
	if p.peek().typ != itemIf {
		p.errorf("expected if, got %s", p.next())
	}
	p.next()

	condition := p.parseExpression(b)
	if !IsBooleanType(condition.Type()) {
		p.errorf("condition is not boolean, but %s", condition.Type().TypeString())
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

	return &IfStatement{label: label, Condition: condition, Statement: stmt, ElseStatement: elseStmt}
}

// parseCaseStatement parses a case statement.
//
//	case-statement =
//		"case" expression "of"
//		case-limb { ";" case-limb } [ ";" ]
//		"end" .
func (p *parser) parseCaseStatement(b *Block, label *string) Statement {
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
		if !labelCompatibleWithType(label, expr.Type()) {
			p.errorf("case label %s doesn't match case expression type %s", label.String(), expr.Type().TypeString())
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
			if !labelCompatibleWithType(label, expr.Type()) {
				p.errorf("case label %s doesn't match case expression type %s", label.String(), expr.Type().TypeString())
			}
		}
		caseLimbs = append(caseLimbs, limb)
	}

	if p.peek().typ != itemEnd {
		p.errorf("expected end, got %s instead", p.peek())
	}
	p.next()

	return &CaseStatement{label: label, Expr: expr, CaseLimbs: caseLimbs}
}

// parseCaseLimb parses a case limb.
//
//	case-limb =
//		case-label-list ":" statement .
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

// parseWithStatement parses a with statement.
//
//	with-statement =
//		"with" record-variable { "," record-variable } "do" statement .
func (p *parser) parseWithStatement(b *Block, label *string) Statement {
	if p.peek().typ != itemWith {
		p.errorf("expected with, got %s instead", p.peek())
	}
	p.next()

	withBlock := &Block{
		Parent:  b,
		Routine: b.Routine,
	}

	var recordExpressions []Expression

	for {

		if p.peek().typ != itemIdentifier {
			p.errorf("expected identifier of record variable, got %s instead", p.peek())
		}

		expr := p.parseExpression(b)

		if !expr.IsVariableExpr() {
			p.errorf("not a variable access expression")
		}

		recType, ok := expr.Type().(*RecordType)
		if !ok {
			p.errorf("variable access not a record type")
		}

		recordExpressions = append(recordExpressions, expr)

		for _, field := range recType.Fields {
			withBlock.Variables = append(withBlock.Variables, &Variable{
				Name:          field.Identifier,
				Type:          field.Type,
				IsRecordField: true,
				BelongsToType: recType,
				//BelongsToVarDecl: varDecl, // TODO: set these correctly.
				//BelongsToParam:   paramDecl,
			})
		}

		if p.peek().typ != itemComma {
			break
		}
		p.next()
	}

	if p.peek().typ != itemDo {
		p.errorf("expected do, got %s instead", p.peek())
	}
	p.next()

	stmt := p.parseStatement(withBlock)

	withBlock.Statements = append(withBlock.Statements, stmt)

	return &WithStatement{
		label:       label,
		RecordExprs: recordExpressions,
		Block:       withBlock,
	}
}

// parseActualParameterList parses an actual parameter list.
//
//	actual-parameter-list =
//		"(" actual-parameter { "," actual-parameter } ")" .
//	actual-parameter =
//		actual-value | actual-variable | actual-procedure | actual-function .
//	actual-value =
//		expression .
//	actual-procedure =
//		procedure-identifier .
//	actual-function =
//		function-identifier .
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

// parseExpression parses an expression.
//
//	expression =
//		simple-expression [ relational-operator simple-expression ] .
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
	if operator == OpIn {
		st, ok := rt.(*SetType)
		if !ok {
			p.errorf("in: expected set type, got %s instead.", rt.TypeString())
		}

		if !lt.IsCompatibleWith(st.ElementType) {
			p.errorf("type %s does not match set type %s", lt.TypeString(), st.ElementType.TypeString())
		}
	} else {
		if !lt.IsCompatibleWith(rt) {
			p.errorf("in relational expression with operator %s, types %s and %s are incompatible", relExpr.Operator, lt.TypeString(), rt.TypeString())
		}
	}

	return relExpr.Reduce()
}

// parseSimpleExpression parses a simple expression.
//
//	simple-expression =
//		[ sign ] term { addition-operator term } .
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
			if !IsBooleanType(simpleExpr.First.Type()) {
				p.errorf("can't use or with %s", simpleExpr.First.Type().TypeString())
			}
		} else {
			if !isIntegerType(simpleExpr.First.Type()) &&
				!isRealType(simpleExpr.First.Type()) &&
				((operator != OperatorAdd && operator != OperatorSubtract) || !isSetType(simpleExpr.First.Type())) {
				p.errorf("can only use %s operator with integer or real types, got %s instead", operator, simpleExpr.First.Type().TypeString())
			}
		}

		nextTerm := p.parseTerm(b)

		if !simpleExpr.First.Type().IsCompatibleWith(nextTerm.Type()) {
			p.errorf("in simple expression involving operator %s, types %s and %s are incompatible", operator, simpleExpr.First.Type().TypeString(), nextTerm.Type().TypeString())
		}

		simpleExpr.Next = append(simpleExpr.Next, &Addition{Operator: operator, Term: nextTerm})

		if !isAdditionOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing simple expression because %s is not an addition operator", p.peek())

	return simpleExpr
}

// parseTerm parses a term.
//
//	term =
//		factor { multiplication-operator factor } .
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
			if !IsBooleanType(term.First.Type()) {
				p.errorf("can't use and with %s", term.First.Type().TypeString())
			}
		case OperatorMultiply:
			if !isIntegerType(term.First.Type()) && !isRealType(term.First.Type()) && !isSetType(term.First.Type()) {
				p.errorf("can only use %s operator with integer, real or set types, got %s instead", operator, term.First.Type().TypeString())
			}
		case OperatorFloatDivide:
			if !isRealType(term.First.Type()) {
				p.errorf("can only use %s operator with real types, got %s instead", operator, term.First.Type().TypeString())
			}
		case OperatorDivide, OperatorModulo:
			if !isIntegerType(term.First.Type()) {
				p.errorf("can only use %s operator with integer types, got %s intead", operator, term.First.Type().TypeString())
			}
		}

		nextFactor := p.parseFactor(b)

		if !term.First.Type().IsCompatibleWith(nextFactor.Type()) {
			p.errorf("in term involving operator %s, types %s and %s are incompatible", operator, term.First.Type().TypeString(), nextFactor.Type().TypeString())
		}

		term.Next = append(term.Next, &Multiplication{Operator: operator, Factor: nextFactor})

		if !isMultiplicationOperator(p.peek().typ) {
			break
		}
	}

	p.logger.Printf("Finished parsing term because %s is not a multiplication operator", p.peek())

	return term
}

// parseFactor parses a factor.
//
//	factor =
//		variable | number | string | set-constructor | "nil" | constant-identifier | bound-identifier | function-designator | "(" expression ")" | "not" factor .
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
				returnType, err := p.validateParameters(funcDecl, params)
				if err != nil {
					p.errorf("function %s: %v", ident, err)
				}
				return &FunctionCallExpr{Name: ident, ActualParams: params, Type_: returnType, FormalParams: funcDecl.FormalParameters}
			}

			if len(funcDecl.FormalParameters) > 0 { // function has formal parameter which are not provided -> it's a functional-parameter
				return &VariableExpr{Name: ident, Type_: &FunctionType{FormalParams: funcDecl.FormalParameters, ReturnType: funcDecl.ReturnType}}
			}
			// TODO: what if function has no formal parameters? is it a functional parameter or a function call? needs resolved later, probably.
			returnType, err := p.validateParameters(funcDecl, []Expression{})
			if err != nil {
				p.errorf("function %s: %v", ident, err)
			}
			return &FunctionCallExpr{Name: ident, Type_: returnType}

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
		se := &StringExpr{decodeStringLiteral(p.next().val)}
		if se.IsCharLiteral() {
			return &CharExpr{Value: se.Value[0]}
		}
		return se
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
		if !IsBooleanType(expr.Type()) {
			p.errorf("can't NOT %s", expr.Type().TypeString())
		}
		return &NotExpr{expr}
	default:
		p.errorf("unexpected %s while parsing factor", p.peek())
	}
	// unreachable
	return nil
}

// parseVariable parses a variable.
//
//	variable =
//		entire-variable | component-variable | referenced-variable .
func (p *parser) parseVariable(b *Block, ident string) Expression {
	var expr Expression

	if paramDecl := b.findFormalParameter(ident); paramDecl != nil {
		expr = &VariableExpr{Name: ident, Type_: paramDecl.Type, ParamDecl: paramDecl}
	} else if procDecl := b.findProcedure(ident); procDecl != nil { // TODO: do we need a separate procedural parameter expression here?
		expr = &VariableExpr{Name: ident, Type_: &ProcedureType{FormalParams: procDecl.FormalParameters}}
	} else if varDecl := b.findVariable(ident); varDecl != nil {
		expr = &VariableExpr{Name: ident, Type_: varDecl.Type, VarDecl: varDecl}
	}

	if expr == nil {
		p.errorf("unknown identifier %s", ident)
	}

	cont := true

	for cont {
		switch p.peek().typ {
		case itemCaret:
			_, isPointerType := expr.Type().(*PointerType)
			_, isFileType := expr.Type().(*FileType)
			if !isPointerType && !isFileType {
				p.errorf("attempting to ^ but expression is not a pointer or file type")
			}
			p.next()
			expr = &DerefExpr{Expr: expr}
		case itemOpenBracket:
			expr = p.parseIndexedVariableExpr(b, expr)
		case itemDot:
			p.next()

			rt, ok := expr.Type().(*RecordType)
			if !ok {
				p.errorf("expression is not a record type, but a %s instead", expr.Type().TypeString())
			}

			if p.peek().typ != itemIdentifier {
				p.errorf("expected identifier, got %s instead", p.peek())
			}
			fieldIdentifier := p.next().val
			field := rt.findField(fieldIdentifier)
			if field == nil {
				p.errorf("unknown field %s", fieldIdentifier)
			}

			expr = &FieldDesignatorExpr{Expr: expr, Field: fieldIdentifier, Type_: field.Type}
		default:
			cont = false
		}
	}

	return expr
}

// parseNumber parses a number.
//
//	number =
//		integer-number | real-number .
//	integer-number =
//		digit-sequence .
//	real-number =
//		digit-sequence "." [ digit-sequence ] [ scale-factor ] |
//		digit-sequence scale-factor .
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

// parseScaleFactor parses a scale factor.
//
//	scale-factor =
//		("E" | "e") [ sign ] digit-sequence .
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

// parseSet parses a set.
//
//	set-constructor =
//		"[" [ member-designator { ", " member-designator } ] "]" .
//	member-designator =
//		expression [ '..' expression ] .
func (p *parser) parseSet(b *Block) *SetExpr {
	if p.peek().typ != itemOpenBracket {
		p.errorf("expected [, found %s instead", p.next())
	}
	p.next()

	set := &SetExpr{}

	if p.peek().typ == itemCloseBracket {
		p.next()
		return set
	}

	expr := p.parseExpression(b)
	if p.peek().typ == itemDoubleDot {
		p.next()
		expr2 := p.parseExpression(b)
		if !expr.Type().Equals(expr2.Type()) {
			p.errorf("when parsing member-designator, lower bound type %s differs from upper bound type %s", expr.Type().TypeString(), expr2.Type().TypeString())
		}
		set.Elements = append(set.Elements, &RangeExpr{LowerBound: expr, UpperBound: expr2})
	} else {
		set.Elements = append(set.Elements, expr)
	}

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
		if p.peek().typ == itemDoubleDot {
			p.next()
			expr2 := p.parseExpression(b)
			if !expr.Type().Equals(expr2.Type()) {
				p.errorf("when parsing member-designator, lower bound type %s differs from upper bound type %s", expr.Type().TypeString(), expr2.Type().TypeString())
			}
			set.Elements = append(set.Elements, &RangeExpr{LowerBound: expr, UpperBound: expr2})
		} else {
			set.Elements = append(set.Elements, expr)
		}
	}

	return set
}

// parseSubExpr parses a sub expression.
//
// "(" expression ")"
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

// parseExpressionList parses an expression list.
//
//	expression-list =
//		expression { "," expression } .
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

// parseArrayType parses an array type.
//
//	array-type =
//		"array" "[ " index-type { "," index-type } "]" "of" element-type .
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

	elementType := p.parseType(b)

	for isArrayType(elementType) {
		arrType := elementType.(*ArrayType)
		indexTypes = append(indexTypes, arrType.IndexTypes...)
		elementType = arrType.ElementType
	}

	return &ArrayType{
		IndexTypes:  indexTypes,
		ElementType: elementType,
		Packed:      packed,
	}
}

// parseSimpleType parses a simple type.
//
//	simple-type =
//		subrange-type | enumerated-type .
func (p *parser) parseSimpleType(b *Block) DataType {
	if p.peek().typ == itemOpenParen {
		return p.parseEnumType(b)
	}

	if typ := b.findType(p.peek().val); typ != nil {
		if _, ok := typ.(*EnumType); ok {
			p.next()
			return typ
		}

		if _, ok := typ.(*SubrangeType); ok {
			p.next()
			return typ
		}
	}

	return p.parseSubrangeType(b)
}

// parseSubrangeType parses a sub-range type.
//
//	subrange-type =
//		lower-bound ".." upper-bound .
//	lower-bound =
//		constant .
//	upper-bound =
//		constant .
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
	case *CharLiteral:
		lowerValue = int(lb.Value)
		typ = charTypeDef.Type
	default:
		p.errorf("expected lower bound to be an integer, an enum value or a char, got a %s instead", lb.ConstantType().TypeString())
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
	case *CharLiteral:
		upperValue = int(ub.Value)
		upperType = charTypeDef.Type
	default:
		p.errorf("expected upper bound to be an integer, an enum value or a char, got a %s instead", ub.ConstantType().TypeString())
	}

	if !upperType.Equals(typ) {
		p.errorf("type of lower bound differs from upper bound: %s vs %s", typ.TypeString(), upperType.TypeString())
	}

	return &SubrangeType{
		LowerBound: lowerValue,
		UpperBound: upperValue,
		Type_:      typ,
	}
}

// parseConstant parses a constant.
//
//	constant =
//		[ sign ] (constant-identifier | number) | string .
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
		sl := &StringLiteral{Value: decodeStringLiteral(p.next().val)}
		if sl.IsCharLiteral() {
			v = &CharLiteral{Value: sl.Value[0]}
		} else {
			v = sl
		}
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

// parseRecordType parses a record type.
//
//	record-type =
//		"record" field-list "end" .
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

// parseFieldList parses a field list.
//
//	field-list =
//		[ (fixed-part [ ";" variant-part ] | variant-part) [ ";" ] ] .
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
		record.VariantField = p.parseVariantPart(b, packed)
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

// parseFixedPart parses the fixed part of a record type.
//
//	fixed-part =
//		record-section { ";" record-section } .
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

// parseVariantPart parses the variant field of a record type.
//
//	variant-part =
//		"case" tag-field type-identifier "of" variant { ";" variant } .
func (p *parser) parseVariantPart(b *Block, packed bool) (field *RecordVariantField) {
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
		Type:     typeDef.Named(typeIdentifier),
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

// parseVariant parses a variant.
//
//	variant =
//		case-label-list ":" "(" field-list ")" .
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

// parseCaseLabelList parses a case label list.
//
//	case-label-list =
//		constant { "," constant } .
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

// parseRecordSection parses a record section.
//
//	record-section =
//		identifier-list ":" type .
func (p *parser) parseRecordSection(b *Block) []*RecordField {
	identifierList := p.parseIdentifierList(b)

	if p.peek().typ != itemColon {
		p.errorf("expected :, got %s instead.", p.peek())
	}
	p.next()

	typ := p.parseType(b)

	var fields []*RecordField

	for _, ident := range identifierList {
		fields = append(fields, &RecordField{Identifier: ident, Type: typ})
	}
	return fields
}

func (p *parser) validateParameters(proc *Routine, actualParams []Expression) (returnType DataType, err error) {
	if proc.validator != nil {
		return proc.validator(actualParams)
	}

	if len(proc.FormalParameters) != len(actualParams) {
		return nil, fmt.Errorf("%d parameter(s) were declared, but %d were provided", len(proc.FormalParameters), len(actualParams))
	}

	for idx := range proc.FormalParameters {
		if !exprCompatible(proc.FormalParameters[idx].Type, actualParams[idx]) {

			return nil, fmt.Errorf("parameter %s expects type %s, but %s was provided",
				proc.FormalParameters[idx].Name, proc.FormalParameters[idx].Type.TypeString(), actualParams[idx].Type().TypeString())
		}

		if proc.FormalParameters[idx].VariableParameter {
			if !actualParams[idx].IsVariableExpr() {
				return nil, fmt.Errorf("parameter %s is a variable parameter, but an actual parameter other than variable was provided",
					proc.FormalParameters[idx].Name)
			}
		}
	}

	return proc.ReturnType, nil
}

// parseIndexedVariableExpr parses an indexed variable.
//
//	indexed-variable =
//		array-variable "[ " expression-list " ]" .
func (p *parser) parseIndexedVariableExpr(b *Block, expr Expression) *IndexedVariableExpr {
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
			p.errorf("string index needs to be an integer type, actually got %s", indexes[0].Type().TypeString())
		}

		return &IndexedVariableExpr{Expr: expr, IndexExprs: indexes, Type_: charTypeDef.Type}
	}

	arrType, ok := expr.Type().(*ArrayType)
	if !ok {
		p.errorf("expression is not array")
	}

	// TODO: support situation where fewer index expressions mean that an array of fewer dimensions is returned.

	if len(indexes) == 0 || len(arrType.IndexTypes) < len(indexes) {
		p.errorf("array has %d dimensions but %d index expressions were provided", len(arrType.IndexTypes), len(indexes))
	}

	for idx, idxType := range arrType.IndexTypes {
		if idx >= len(indexes) {
			break
		}
		if !idxType.IsCompatibleWith(indexes[idx].Type()) {
			fmt.Printf("idxType = %s index[idx].Type = %s\n", spew.Sdump(idxType), spew.Sdump(indexes[idx].Type()))
			p.errorf("array dimension %d is of type %s, but index expression type %s was provided", idx, idxType.TypeString(), indexes[idx].Type().TypeString())
		}
	}

	return &IndexedVariableExpr{Expr: expr, IndexExprs: indexes, Type_: arrType.ElementType}
}

// parseWrite parses a write statement.
//
// TODO: add EBNF.
func (p *parser) parseWrite(b *Block, ln bool, label *string) *WriteStatement {
	if p.peek().typ != itemOpenParen {
		p.errorf("expected (, got %s instead", p.peek())
	}
	p.next()

	stmt := &WriteStatement{label: label, AppendNewLine: ln}

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
		p.errorf("decimal places format is not allowed for type %s", expr.Type().TypeString())
	}

	return widthExpr, decimalPlacesExpr
}

func (p *parser) verifyWriteType(typ DataType, ln bool) {
	funcName := "write"
	if ln {
		funcName += "ln"
	}

	allowedWriteTypes := []DataType{&IntegerType{}, &RealType{}, charTypeDef.Type, &StringType{}, booleanTypeDef.Type}

	for _, at := range allowedWriteTypes {
		if at.Equals(typ) {
			return
		}
	}

	if isOrdinalType(typ) {
		return
	}

	// arrays of char are also allowed.
	if isCharArray(typ) {
		return
	}

	p.errorf("can't use variables of type %s with %s", typ.TypeString(), funcName)
}
