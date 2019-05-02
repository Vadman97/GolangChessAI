package ai

func MaxScore(a, b Value) (result Value) {
	if b > a {
		result = b
	} else {
		result = a
	}
	return
}

func MinScore(a, b Value) (result Value) {
	if b < a {
		result = b
	} else {
		result = a
	}
	return
}
