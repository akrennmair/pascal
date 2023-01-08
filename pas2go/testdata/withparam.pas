program test;

type foox = record
			a : integer;
			b : real
		end;
	fooy = record
		z : record
			c : integer
		end
	end;

var xx : foox;
	yy : fooy;

procedure quux(x : foox; y : fooy);
begin
	with x, y do
	begin
		a := 42;
		b := 23.5;
		writeln('a = ', a);
		writeln('b = ', b);
		with z do
		begin
			c := 9001;
			writeln('c = ', c)
		end
	end
end;

begin
	quux(xx, yy)
end.
