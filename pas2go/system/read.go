package system

import "fmt"

// TODO: improve this and make I/O closer to Pascal.

func Read(a ...any) {
	fmt.Scanln(a...)
}

func Readln(a ...any) {
	fmt.Scanln(a...)
}
