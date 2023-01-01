package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParserSuccesses(t *testing.T) {
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
		{
			"procedure declaration where formal parameter is used in procedure body",
			`program test;

			procedure quux(a : string);
			begin
				writeln(a)
			end;

			begin
				quux('hello world')
			end.
			`,
		},
		{
			"function declaration where formal parameter is used in function body and function return value is assigned",
			`program test;

			function square(a : integer) : integer;
			begin
				square := a * a
			end;

			begin
				writeln(square(2))
			end.
			`,
		},
		{
			"const declaration of a string constant",
			`program test;

			const foo = 'hello world';

			begin
				writeln(foo)
			end.`,
		},
		{
			"const declaration of real constants",
			`program test;

			const foo = 3.1415;
				bar = -0.1;

			begin
				writeln(foo, bar)
			end.`,
		},
		{
			"const declaration of a real constant and a negated version referring to the first one",
			`program test;

			const foo = 3.1415;
				bar = -foo;

			begin
				if bar < foo then
					writeln('bar is indeed smaller than foo')
			end.`,
		},
		{
			"type and variable declaration with file type",
			`program test;

			type foo = file of real;

			var bar : file of record
								a, b : integer;
								c, d : real
							end;
			begin
			end.`,
		},
		{
			"subrange type of two positive integers",
			`program test;

			type foo = 1..100;

			begin
			end.`,
		},
		{
			"subrange type of two constants",
			`program test;

			const min = 1;
				max = 100;

			type foo = min..max;

			begin
			end.`,
		},
		{
			"subrange type of negative and positive integers",
			`program test;

			type foo = -1..+3;

			begin
			end.`,
		},
		{
			"subrange type of negative and positive constant",
			`program test;

			const x = 23;

			type foo = -x..x;

			begin
			end.`,
		},
		{
			"record type with only variant part",
			`program test;

			type foo = record
			case xxx : integer of
			1 : (a : string);
			2 : (b : integer);
			3, 4 : (c : real; d : string)
			end;

			begin
			end.`,
		},
		{
			"record type with fixed part and variant part",
			`program test;

			type foo = record
				quux : integer;
				case xxx : integer of
				1 : (a : string);
				2 : (b : integer);
				3, 4 : (c : real; d : string)
			end;

			begin
			end.`,
		},
		{
			"record type with variant parts with empty field list and nested variant part",
			`program test;

			type foo = record
				quux : integer;
				case xxx : integer of
				1 : ();
				2 : (case yyy : integer of
					3: (a : integer);
					4: (b : real)
				);
				3 : (
					bla : real;
					case zzz : integer of
					5: (c : integer);
					6: (d : string)
				)
			end;

			begin
			end.`,
		},
		{
			"record type with fixed parts and semicolon after last field",
			`program test;

			type foo = record
				quux : integer;
				bla: string;
			end;

			begin
			end.`,
		},
		{
			"record type with only variant part and semicolon after last field",
			`program test;

			type foo = record
			case xxx : integer of
			1 : (a : string);
			2 : (b : integer);
			3, 4 : (c : real; d : string);
			end;

			begin
			end.`,
		},
		{
			"case statement",
			`program test;

			var x : integer;

			begin
				case x of
				1 : writeln('hello world');
				2 : writeln('goodbye world')
				end
			end.`,
		},
		{
			"case statement with optional semicolon after last case limb",
			`program test;

			var x : integer;

			begin
				case x of
				1 : writeln('hello world');
				2 : writeln('goodbye world');
				end
			end.`,
		},
		{
			"with statement",
			`program test;

			var x : record
				a : integer;
				b : real;
			end;

			begin
				with x do
				begin
					a := 23;
					b := 23.5
				end
			end.`,
		},
		{
			"with statement that also uses other variables",
			`program test;

			var x : record
					a : integer;
					b : real;
				end;
				y : integer;

			begin
				with x do
				begin
					a := 23;
					b := 23.5;
					y := 42
				end
			end.`,
		},
		{
			"nested with statements and record-identifier.field-identifier syntax of nested records",
			`program test;

			var y : record
					a : integer;
					b : record
						c : real;
						d : string;
					end;
				end;

			begin
				with y do
				begin
					a := 23; { addresses y.a }
					b.c := 23.5; { addresses y.b.c }
					with b do
					begin
						d := 'hello' { addresses y.b.d }
					end
				end
			end.`,
		},
		{
			"string constant",
			`program test;

			const hello = 'hello world';

			begin
				if hello = 'goodbye' then
					writeln(hello)
			end.`,
		},
		{
			"enum type and symbols being used",
			`program test;

			type cards = (clubs, diamonds, hearts, spades);

			var a : cards;

			begin
				a := clubs;
				if a <> diamonds then
					writeln('a is not diamonds')
			end.`,
		},
		{
			"variable of enum type directly used",
			`program test;

			var a : (clubs, diamonds, hearts, spades);

			begin
				a := clubs;
				if a <> diamonds then
					writeln('a is not diamonds')
			end.`,
		},
		{
			"variable of enum type directly used",
			`program test;

			var a : (clubs, diamonds, hearts, spades);

			procedure foo;
			begin
				if a <> diamonds then
					writeln('a is not diamonds')
			end;

			begin
				a := clubs;
				foo
			end.`,
		},
		{
			"more complex data structures with enum types",
			`program test;

			const king = 13;
				queen = 12;
				jack = 11;

			type cardType = (clubs, diamonds, hearts, spades);
				allCards = array [2..king] of cardType;

			var myCards : allCards;

			procedure printCard(card : cardType);
			begin
				case card of
				clubs : writeln('clubs');
				diamonds : writeln('diamonds');
				hearts : writeln('hearts');
				spades : writeln('spades');
				end
			end;

			procedure printCards(cardSet : allCards);
			var i : integer;
			begin
				for i := 2 to king do
					printCard(cardSet[i])
			end;

			begin
				myCards[king] := spades;
				myCards[3] := hearts;
				printCards(myCards)
			end.`,
		},
		{
			"with statement of formal parameter",
			`program test;

			type foo = record
					a : integer;
					b : real;
				end;

			var x : foo;

			procedure quux(my : foo);
			begin
				with my do
				begin
					a := 23;
					b := 3.1415
				end
			end;

			begin
				quux(x)
			end.`,
		},
		{
			"with statement of formal parameter that also uses variable used in with",
			`program test;

			type foo = record
					a : integer;
					b : real;
				end;

			var x : foo;

			procedure quux(my : foo);
			begin
				with my do
				begin
					a := 23;
					b := 3.1415;
					my.a := 42
				end
			end;

			begin
				quux(x)
			end.`,
		},
		{
			"procedure with variable parameter",
			`program test;

			var x : integer;

			procedure quux(var y : integer);
			begin
				y := 3
			end;

			begin
				x := 2;
				writeln('before: ', x);
				quux(x);
				writeln('after: ', x)
			end.`,
		},
		{
			"procedure with variable parameter inside sub expression",
			`program test;

			var x : integer;

			procedure quux(var y : integer);
			begin
				y := 3
			end;

			begin
				x := 2;
				writeln('before: ', x);
				quux((x));
				writeln('after: ', x)
			end.`,
		},
		{
			"procedural parameter",
			`program test;

			procedure a(procedure b(c : integer); i : integer);
			begin
				writeln('foo');
				b(i);
				writeln('bar')
			end;
			
			procedure printInt(i : integer);
			begin
				writeln('i = ', i)
			end;
			
			begin
				a(printInt, 23)
			end.`,
		},
		{
			"functional parameter",
			`program test;

			procedure a(function transform(c : integer) : integer; i : integer);
			begin
				writeln(i, ' -> ', transform(i))
			end;
			
			function square(i : integer) : integer;
			begin
				square := i * i
			end;
			
			function times2(i : integer) : integer;
			begin
				times2 := i * 2
			end;
			
			begin
				a(square, 23);
				a(times2, 42)
			end.`,
		},
		{
			"dereferencing a pointer",
			`program test;

			type tint = ^integer;

			var y : tint;
			
			begin
				y^ := 23
			end.`,
		},
		{
			"dereferencing a record and using a field",
			`program test;

			type foo = record
						a : integer;
						b : integer
					end;
				pfoo = ^foo;


			var y : pfoo;
			
			begin
				y^.a := y^.b
			end.`,
		},
		{
			"dereferencing a record and using a field, nested",
			`program test;

			type foo = record
						a : integer;
						b : ^integer
					end;
				pfoo = ^foo;
				fooarray = array[1..10] of pfoo;
				pfooarray = ^fooarray;
				bar = record
						q : pfooarray;
						r : ^integer
					end;
				pbar = ^bar;

			var y : pbar;

			begin
				y^.q^[1]^.a := y^.r^;
				y^.q^[1]^.b^ := y^.r^
			end.`,
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

func TestParserErrors(t *testing.T) {
	testData := []struct {
		Name          string
		ExpectedError string
		Code          string
	}{
		{
			"program doesn't end with .",
			`expected ., got ";" instead`,
			`program test;
			begin end;`,
		},
		{
			"program doesn't start with program",
			`expected program, got "for"`,
			`for test; begin end.`,
		},
		{
			"program isn't defined by identifier",
			`expected identifier, got "23"`,
			"program 23; begin end.",
		},
		{
			"program header isn't terminated by semicolon",
			`expected semicolon, got "."`,
			"program test. begin end.",
		},
		{
			"statement part doesn't start with begin",
			`expected begin, got "for" instead`,
			"program test; for end.",
		},
		{
			"declaration label is not an unsigned digit sequence",
			`expected number, got "foo"`,
			"program test; label foo; begin end.",
		},
		{
			"label declaration is improperly terminated",
			`expected comma or semicolon, got "."`,
			"program test; label 123. begin end.",
		},
		{
			"type declaration with unknown type",
			`unknown type unknown`,
			"program test; type foo = unknown; begin end.",
		},
		{
			"type declaration with pointer to integer literal",
			`expected type after ^, got "123"`,
			"program test; type foo = ^123; begin end.",
		},
		{
			"type declaration with two packed keywords",
			`packed can only be used with array, record, set or file, found "packed" instead`,
			"program test; type foo = packed packed array[1..10] of integer; begin end.",
		},
		{
			"set type declaration with missing of keyword",
			`expected of after set, got "integer"`,
			"program test; type foo = set integer; begin end.",
		},
		{
			"integer literal as type in type definition",
			`expected .., got ";"`,
			"program test; type foo = 234; begin end.",
		},
		{
			"enum type declaration with improper termination",
			`expected ), got ";"`,
			"program test; type foo = (bar, quux;); begin end.",
		},
		{
			"enum type declaration with integer literal",
			`expected identifier, got "456"`,
			"program test; type foo = (456, bar, quux); begin end.",
		},
		{
			"enum type declaration with integer literal (2)",
			`expected identifier, got "678"`,
			"program test; type foo = (bar, 678, quux); begin end.",
		},
		{
			"var declaration without :",
			`expected :, got "="`,
			"program test; var foo = integer; begin end.",
		},
		{
			"var declaration incorrectly terminated",
			`expected ;, got ":"`,
			"program test; var foo : integer: begin end.",
		},
		{
			"procedure heading incorrectly terminated",
			`expected ;, got ":"`,
			"program test; procedure foo: begin end.",
		},
		{
			"function heading incorrectly terminated",
			`expected ;, got ":"`,
			"program test; procedure foo : integer: begin end.",
		},
		{
			"procedure declaration incorrectly terminated",
			`expected ;, got "^"`,
			"program test; procedure foo; begin end^",
		},
		{
			"function declaration incorrectly terminated",
			`expected ;, got "."`,
			"program test; function foo : integer; begin end.",
		},
		{
			"duplicate label declaration",
			`duplicate label identifier "123"`,
			`program test;
			label 123, 123;
			begin end.
			`,
		},
		{
			"duplicate const declaration",
			`duplicate const identifier "foo"`,
			`program test;
			const
				foo = 123;
				foo = 234;
			begin end.
			`,
		},
		{
			"duplicate type definition",
			`duplicate type name "foo"`,
			`program test;
			type
				foo = integer;
				foo = array[1..10] of integer;
			begin end.
			`,
		},
		{
			"type definition that's already in use for a const",
			`duplicate type name "foo"`,
			`program test;
			const foo = 123;
			type foo = integer;
			begin end.
			`,
		},
		{
			"record type with only fixed part and duplicate field names",
			"duplicate field name a",
			`program test;

			type foo = record
				a : integer;
				a : string
			end;

			begin
			end.`,
		},
		{
			"record type with only variant part and duplicate field names",
			"duplicate variant field name a",
			`program test;

			type foo = record
			case xxx : integer of
			1 : (a : string);
			2 : (a : integer)
			end;

			begin
			end.`,
		},
		{
			"record type with fixed part and variant part and duplicate field names",
			"duplicate variant field name quux",
			`program test;

			type foo = record
				quux : integer;
				case xxx : integer of
				1 : (quux : string);
				2 : (b : integer)
			end;

			begin
			end.`,
		},
		{
			"record type with variant parts with empty field list and nested variant part and duplicate field names",
			"duplicate variant field name bla",
			`program test;

			type foo = record
				quux : integer;
				case xxx : integer of
				1 : ();
				2 : (case yyy : integer of
					3: (a : integer);
					4: (b : real)
				);
				3 : (
					bla : real;
					case zzz : integer of
					5: (bla : integer);
					6: (d : string)
				)
			end;

			begin
			end.`,
		},

		{
			"left expression not followed by assignment",
			`unexpected token "writeln" in statement`,
			`program test;
			var x : integer;
			begin
				x writeln('hello world')
			end.`,
		},
		{
			"procedure with variable parameter inside sub expression",
			"parameter y is a variable parameter, but an actual parameter other than variable was provided",
			`program test;

			var x : integer;

			procedure quux(var y : integer);
			begin
				y := 3
			end;

			begin
				x := 2;
				writeln('before: ', x);
				quux(x * x);
				writeln('after: ', x)
			end.`,
		},
		{
			"dereferencing a record and using a field, nested, but it's attempting to dereference an integer field",
			"attempting to ^ but expression is not a pointer type",
			`program test;

			type foo = record
						a : integer;
						b : ^integer
					end;
				pfoo = ^foo;
				fooarray = array[1..10] of pfoo;
				pfooarray = ^fooarray;
				bar = record
						q : pfooarray;
						r : ^integer
					end;
				pbar = ^bar;

			var y : pbar;

			begin
				y^.q^[1]^.a^ := y^.r^
			end.`,
		},
	}

	for idx, tt := range testData {
		t.Run(tt.Name, func(t *testing.T) {
			p, err := parse(fmt.Sprintf("test_%d.pas", idx), tt.Code)
			if err == nil {
				t.Errorf("Parsing code unexpectedly didn't return error")
			} else if !strings.Contains(err.Error(), tt.ExpectedError) {
				t.Logf("expected error = %s", tt.ExpectedError)
				t.Errorf("Parsing returned error, but didn't contain expected error message")
			}
			_ = p
			t.Logf("error = %v", err)
		})
	}
}
