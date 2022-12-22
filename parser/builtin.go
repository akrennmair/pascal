package parser

func findBuiltinProcedure(name string) *procedure {
	for _, proc := range builtinProcedures {
		if proc.Name == name {
			return proc
		}
	}

	return nil
}

var builtinProcedures = []*procedure{
	{Name: "writeln"},
}
