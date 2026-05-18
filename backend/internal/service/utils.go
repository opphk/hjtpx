package service

import "time"

func ParseDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

func ParseDurationMs(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
