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
