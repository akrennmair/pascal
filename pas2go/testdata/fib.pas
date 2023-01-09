program fib;

var max : integer;
	fib1, fib2 : integer;
	nextNum : integer;


begin
	write('generating fibonacci numbers to what maximum? ');
	readln(max);

	fib1 := 1;
	fib2 := 1;

	writeln(fib1);

	while fib2 < max do
	begin
		writeln(fib2);
		nextNum := fib1 + fib2;
		fib1 := fib2;
		fib2 := nextNum
	end
end.
