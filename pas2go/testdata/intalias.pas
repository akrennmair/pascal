program test;

type pt = ^intalias;
	intalias = char;

var p : pt;

begin
	new(p)
end.
