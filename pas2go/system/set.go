package system

type SetType[T comparable] struct {
	values []T
}

func (ts *SetType[T]) In(elem T) bool {
	for _, v := range ts.values {
		if v == elem {
			return true
		}
	}
	return false
}

func (ts *SetType[T]) Union(o SetType[T]) SetType[T] {
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

func (ts *SetType[T]) Difference(o SetType[T]) SetType[T] {
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

func (ts *SetType[T]) Intersection(o SetType[T]) SetType[T] {
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

func Set[T comparable](values ...T) SetType[T] {
	return SetType[T]{values: values}
}
