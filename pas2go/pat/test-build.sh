#!/bin/sh

mkdir -p test
cp iso7185pat.out test/x.go
go build -o test/x test/x.go
