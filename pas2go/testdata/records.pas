program test;

type foo = record
		b : integer;
		c : real;
	end;
	bar = record
		d : string;
		e : foo;
	end;

var x : bar;

begin
	x.d := 'hello';
	x.e.b := 42;
	x.e.c := 3.1415;
	writeln(x.d, x.e.b, x.e.c)
end.
