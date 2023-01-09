program ex42a;

var i : integer;
	h, t, o : integer;

function fac(i : integer) : integer;
begin
	if i <= 1 then
		fac := 1
	else
		fac := i * fac(i - 1)
end;

begin
	for i:=100 to 999 do
	begin
		h := i div 100;
		t := (i mod 100) div 10;
		o := i mod 10;
		if i = (fac(h) + fac(t) + fac(o)) then
			writeln(i, ' = ', h, '! + ', t, '! + ', o, '!')
	end
end.
