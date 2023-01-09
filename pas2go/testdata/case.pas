program test;

label 10;

var i : integer;

begin
	i := 4;
	10:
	case i of
	0: writeln('goodbye world');
	1: writeln('hello world');
	2, 3, 4:
		begin
			writeln('wait for it...');
			i := i - 1;
			goto 10
		end;
	end
end.
