package system

type SetType[T comparable] struct {
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

func Range[T comparable](from, to T) SetType[T] {
	set := SetType[T]{}
	// TODO: implement
	return set
}

func Set[T comparable](values ...T) SetType[T] {
	return SetType[T]{values: values}
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
