program test;

var b : integer;

procedure x(var a : integer);
begin
	a := 1
end;

begin
	x(b);
	writeln('b = ', b)
end.
