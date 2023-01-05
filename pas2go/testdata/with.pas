program test;

var x : record
		a : integer;
		b : real
	end;

	y : record
		z : record
			c : integer
		end
	end;

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

end.
