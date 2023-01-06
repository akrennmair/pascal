{$mode ISO}
program test;

function factorial(i : integer) : integer;
begin
    if i = 1 then
        factorial := 1
    else
        factorial := i * factorial(i - 1)
end;

begin
    writeln('10! = ', factorial(10))
end.
