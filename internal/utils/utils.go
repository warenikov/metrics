package utils

import "strconv"

func Float64ToString(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func Int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
