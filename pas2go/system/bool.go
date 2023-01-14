package system

func BoolPred(b bool) bool {
	if b {
		return false
	}
	return true
}

func BoolSucc(b bool) bool {
	return true
}

func BoolOrd(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	if i == 0 {
		return false
	}
	return true
}

func BoolRange(from bool, to bool) (list []bool) {
	for i := BoolOrd(from); i <= BoolOrd(to); i++ {
		list = append(list, intToBool(i))
	}
	return list
}

func BoolRangeDown(from bool, to bool) (list []bool) {
	for i := BoolOrd(from); i <= BoolOrd(to); i-- {
		list = append(list, intToBool(i))
	}
	return list
}
