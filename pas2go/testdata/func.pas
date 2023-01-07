{$mode ISO}
program test;

procedure a(function b(c : integer) : integer; i : integer);
begin
	writeln(i, ' -> ', b(i))
end;

function times2(i : integer) : integer;
begin
	times2 := i * 2
end;

function square(i : integer) : integer;
begin
	square := i * i
end;

begin
	a(times2, 23);
	a(square, 42)
end.
