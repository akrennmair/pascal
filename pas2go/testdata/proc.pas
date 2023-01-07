program test;

procedure a(procedure b(c : integer); i : integer);
begin
        writeln('foo');
        b(i);
        writeln('bar')
end;

procedure printInt(i : integer);
begin
        writeln('i = ', i)
end;

begin
        a(printInt, 23)
end.
