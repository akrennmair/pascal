package parser

import "fmt"

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
		return FindBuiltinProcedure(name)
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
		return FindBuiltinFunction(name)
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
