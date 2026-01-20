package utils

func DefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func DefaultInt64(v, def int64) int64 {
	if v == 0 {
		return def
	}
	return v
}

func DefaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func DefaultFloat64(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

func DefaultBool(v, def bool) bool {
	if v == false {
		return def
	}
	return v
}
