package system

import (
	"fmt"
	"reflect"
)

type setTypeConstraint interface {
	byte | ~int | bool
}

type intSetTypeConstraint interface {
	byte | ~int
}

type SetType[T setTypeConstraint] struct {
	values []T
}

func (ts SetType[T]) In(elem T) bool {
	for _, v := range ts.values {
		if v == elem {
			return true
		}
	}
	return false
}

func (ts SetType[T]) Union(o SetType[T]) SetType[T] {
	set := make(map[T]struct{})

	for _, v := range ts.values {
		set[v] = struct{}{}
	}
	for _, v := range o.values {
		set[v] = struct{}{}
	}

	newSet := SetType[T]{}

	for k := range set {
		newSet.values = append(newSet.values, k)
	}

	return newSet
}

func (ts SetType[T]) Difference(o SetType[T]) SetType[T] {
	set := make(map[T]struct{})

	for _, v := range ts.values {
		set[v] = struct{}{}
	}
	for _, v := range o.values {
		delete(set, v)
	}

	newSet := SetType[T]{}

	for k := range set {
		newSet.values = append(newSet.values, k)
	}

	return newSet
}

func (ts SetType[T]) Intersection(o SetType[T]) SetType[T] {
	set := make(map[T]struct{})

	for _, v := range ts.values {
		set[v] = struct{}{}
	}

	newSet := SetType[T]{}

	for _, v := range o.values {
		if _, ok := set[v]; ok {
			newSet.values = append(newSet.values, v)
		}
	}

	return newSet
}

func Range[T intSetTypeConstraint](from, to T) []T {
	var values []T
	for i := from; i <= to; i++ {
		values = append(values, i)
	}
	return values
}

func Set[T setTypeConstraint](values ...any) SetType[T] {
	set := SetType[T]{}
	for _, v := range values {
		switch vv := v.(type) {
		case T:
			set.values = append(set.values, vv)
		case []T:
			set.values = append(set.values, vv...)
		default:
			var foo T
			panic(fmt.Errorf("can't construct set[%T] from %T", foo, v))
		}
	}
	return set
}

func (ts SetType[T]) Equals(os SetType[T]) bool {
	// this is not very efficient.
	for _, v := range ts.values {
		if !os.In(v) {
			return false
		}
	}

	for _, v := range os.values {
		if !ts.In(v) {
			return false
		}
	}

	return true
}

func (ts SetType[T]) NotEquals(os SetType[T]) bool {
	return !ts.Equals(os)
}

func (ts SetType[T]) Less(os SetType[T]) bool {
	// TODO: implement
	return false
}

func (ts SetType[T]) LessEqual(os SetType[T]) bool {
	// TODO: implement
	return false
}

func (ts SetType[T]) Greater(os SetType[T]) bool {
	// TODO: implement
	return false
}

func (ts SetType[T]) GreaterEqual(os SetType[T]) bool {
	// TODO: implement
	return false
}

func SetAssign[T1, T2 intSetTypeConstraint](to *SetType[T1], from SetType[T2]) {
	to.values = nil
	for _, v := range from.values {
		to.values = append(to.values, assignConvert[T1](v))
	}
}

func SetAssignFromBool[T1 intSetTypeConstraint](to *SetType[T1], from SetType[bool]) {
	to.values = nil
	for _, v := range from.values {
		var x T1
		if v {
			x = T1(1)
		}
		to.values = append(to.values, x)
	}
}

func SetAssignToBool[T1 intSetTypeConstraint](to *SetType[bool], from SetType[T1]) {
	to.values = nil
	for _, v := range from.values {
		var x bool
		if v != 0 {
			x = true
		}
		to.values = append(to.values, x)
	}
}

func BoolSetAssign(to *SetType[bool], from SetType[bool]) {
	to.values = nil
	for _, v := range from.values {
		to.values = append(to.values, v)
	}
}

func assignConvert[T1, T2 intSetTypeConstraint](v T2) T1 {
	var b bool
	if vv := reflect.ValueOf(v); vv.Type() == reflect.TypeOf(b) {
		b := vv.Bool()
		if b {
			return T1(1)
		}
	}
	return T1(v)
}
