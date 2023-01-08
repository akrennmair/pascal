program test;

type foox = record
			a : integer;
			b : real
		end;

var xx : foox;

procedure quux(var x : foox);
begin
	with x do
	begin
		a := 42;
		b := 23.5
	end
end;

begin
	quux(xx);
	writeln('xx.a = ', xx.a);
	writeln('xx.b = ', xx.b)
end.
