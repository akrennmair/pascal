program test;

type foo = record
		b : integer;
		c : real;
	end;
	bar = record
		d : string;
		e : foo;
	end;

var y : bar;

procedure quux(var x : bar);
begin
	x.d := 'hello';
	x.e.b := 42;
	x.e.c := 3.1415
end;

begin
	quux(y);
	writeln(y.d, y.e.b, y.e.c)
end.
