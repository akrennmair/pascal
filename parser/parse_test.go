package parser

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParser(t *testing.T) {
	testData := []struct {
		name string
		code string
	}{
		{
			"empty",
			`program test;
			begin
			end.`,
		},
		{
			"label declaration",
			`program test;
			label 1;
			begin
			end.`,
		},
		{
			"multiple label declarations",
			`program test;
			label 1, 2, 3;
			begin
			end.`,
		},
		{
			"const declaration",
			`program test;
			const foo = 1;
			begin
			end.`,
		},
		{
			"multiple const declarations",
			`program test;
			const
				foo = 1;
				bar = 2;
				quux = 3;
			begin
			end.`,
		},
		{
			"type declaration alias",
			`program test;
			type bar = integer;
			begin
			end.`,
		},
		{
			"type declaration pointer",
			`program test;
			type foo = ^integer;
			begin
			end.`,
		},
		{
			"type declaration enumerated",
			`program test;
			type foo = ( bar );
			begin
			end.`,
		},
		{
			"type declaration enumerated 2",
			`program test;
			type foo = ( bar, baz, quux );
			begin
			end.`,
		},
		{
			"type declaration set",
			`program test;
			type foo = set of integer;
			begin
			end.`,
		},
		{
			"type declaration set of pointers",
			`program test;
			type foo = set of ^integer;
			begin
			end.`,
		},
		{
			"type declaration set of enumerated",
			`program test;
			type foo = set of ( bar, baz, quux );
			begin
			end.`,
		},
		{
			"type declaration files",
			`program test;
			type foo = file of integer;
			bar = file of set of integer;
			begin
			end.`,
		},
		{
			"type declaration packed set",
			`program test; type foo = packed set of integer; begin end.`,
		},
		{
			"type declaration integer variable",
			`program test;
			var foo : integer;
			begin
			end.`,
		},
		{
			"type declaration multiple sets of integers, and two reals",
			`program test;
			var foo, bar, baz : set of integer;
				quux, bla : real;
			begin
			end.`,
		},
		{
			"empty procedure",
			`program test;
			procedure foo;
			begin
			end;
			begin
			end.`,
		},
		{
			"procedure with variables",
			`program test;
			procedure foo;
				var bar : integer;
			begin
			end;
			begin
			end.`,
		},
		{
			"procedure with 1 parameter",
			`program test;
			procedure foo(a : integer);
			begin
			end;
			begin
			end.`,
		},
		{
			"procedure with 2 parameters",
			`program test;
			procedure foo(a, b : integer);
			begin
			end;
			begin
			end.`,
		},
		{
			"procedure with multiple parameters, including enum",
			`program test;
			procedure foo(a, b : integer; c, d : ^integer; e : ( bla, oops ));
			begin
			end;
			begin
			end.`,
		},
		{
			"procedure with var parameters",
			`program test;
			procedure foo(var a, b : integer);
			begin
			end;
			begin
			end.`,
		},
		{
			"function returning integer",
			`program test;
			function foo : integer;
			begin
			end;
			begin
			end.`,
		},
		{
			"function with 1 parameter",
			`program test;
			function foo(a : integer) : integer;
			begin
			end;
			begin
			end.`,
		},
		{
			"function with multiple parameters",
			`program test;
			function foo(a, b : real; var c, d : integer) : string;
			begin
			end;
			begin
			end.`,
		},
		{
			"program with label declarations and gotos",
			`program test;
			label 23, 42;
			begin
				23: goto 42;
				42: goto 23
			end.`,
		},
		{
			"program with assignments",
			`program test;
				var a, b : integer;
			begin
				a := 3;
				b := a
			end.`,
		},
		{
			"while loop with begin end block",
			`program test;
			var b : boolean;
				a : integer;
			begin
				while b do
				begin
					a := 3
				end
			end.`,
		},
		{
			"for loop",
			`program test;
			var i : integer;
			begin
				for i := 0 to 23 do
					writeln(i)
			end.`,
		},
		{
			"repeat until",
			`program test;
			var a : boolean;
			begin
				repeat
					writeln(23)
				until a
			end.`,
		},
		{
			"simple if",
			`program test;
			var a : integer;
			begin
				if a <> 24 then
					writeln(23)
			end.`,
		},
		{
			"if else",
			`program test;
			var a : boolean;
			begin
				if a then
					writeln(23)
				else
					writeln(42)
			end.`,
		},
		{
			"if else with begin end",
			`program test;
			var a : string;
			begin
				if a <> '' then
				begin
					writeln(23)
				end
				else
				begin
					writeln(42)
				end
			end.`,
		},
		{
			"procedure call with multiple actual parameters",
			`program test;
			begin
				writeln('The answer is ', 42)
			end.`,
		},
		{
			"variable with one-dimensional integer array",
			`program test;
			var a : array[1..10] of integer;
			begin
			end.
			`,
		},
		{
			"variable with two-dimensional integer array",
			`program test;
			var a : array[1..10, -1..+1] of integer;
			begin
			end.
			`,
		},
		{
			"variable with one-dimenstional integer array assignment",
			`program test;
			var a : array[1..10] of integer;
			begin
				a[1] := 3;
				a[2] := 4
			end.
			`,
		},
		{
			"variable with one-dimenstional integer array and constants as subrange",
			`program test;
			const min = -10;
				max = 10;
			var a : array[min..max] of integer;
			begin
				a[1] := 3;
				a[2] := 4
			end.
			`,
		},
		{
			"variable with one-dimenstional integer array and negated constant in subrange",
			`program test;
			const size = 10;
			var a : array[-size..+size] of integer;
			begin
				a[1] := 3;
				a[2] := 4
			end.
			`,
		},
		{
			"record type definition",
			`program test;
			type foo = record
						a, b : integer;
						c, d : string
					end;
			begin
			end.
			`,
		},
		{
			"record variable definition",
			`program test;
			var foo : record
						a, b : integer;
						c, d : string
					end;
			begin
			end.
			`,
		},
		{
			"record variable definition and field assignment",
			`program test;
			var foo : record
						a : integer
					end;
			begin
				foo.a := 3
			end.
			`,
		},
		{
			"function declaration, then call of function in condition",
			`program test;
			const a = 42;
			function x : integer;
			begin
			end;

			begin
				if x <> a then
					writeln(23)
			end.
			`,
		},
		{
			"procedure declaration with no parameters, then call of procedure in main program",
			`program test;
			const a = 42;
			procedure x;
			begin
				writeln(a)
			end;

			begin
				x
			end.
			`,
		},
		{
			"type declaration, then type used in variable",
			`program test;
			type foo = array[1..10] of integer;
			var x : foo;
			begin
				x[1] := 0
			end.
			`,
		},
		{
			"type declaration of alias, then alias type used in variable",
			`program test;
			type foo = array[1..10] of integer;
				bar = foo;
			var x : bar;
			begin
				if x[1] <> 0 then
					writeln(23)
			end.
			`,
		},
		{
			"type declaration, then type is used in variable in procedure",
			`program test;

			type foo = array[1..10] of integer;

			procedure quux;
			var x : foo;
			begin
			if x[1] <> 0 then
				writeln(23)
			end;

			begin
				quux
			end.
			`,
		},
	}

	for idx, testEntry := range testData {
		t.Run(testEntry.name, func(t *testing.T) {
			p, err := parse(fmt.Sprintf("test_%d.pas", idx), testEntry.code)
			if err != nil {
				t.Errorf("%d. parse failed: %v", idx, err)
			}
			t.Logf("p = %s", spew.Sdump(*p))
		})
	}
}
