package utils

import "time"

func ToSecond(t int) time.Duration {
	return time.Duration(t) * time.Second
}

func ToMinute(t int) time.Duration {
	return time.Duration(t) * time.Minute
}
