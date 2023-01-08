#!/bin/bash -ex

for f in *.golden ; do
	cp $f x.go
	go build -o x x.go || ( echo "Failed to compile $f" ; exit 1 )
	rm x x.go
done
