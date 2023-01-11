program test;

var x : ^boolean;

begin
	new(x);
	x^ := true;
	dispose(x)
end.
