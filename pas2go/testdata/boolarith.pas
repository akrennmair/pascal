program test;

var a, b : boolean;

begin
	a := true;
	b := false;
	writeln('boolean succ(true): ', succ(a));
	writeln('boolean succ(false): ', succ(b));
	writeln('boolean succ(succ(false)): ', succ(succ(b)));
	writeln('boolean pred(true): ', pred(a));
	writeln('boolean pred(false): ', pred(b));
	writeln('boolean pred(pred(false)): ', pred(pred(b)));
end.
