package utils

import "time"

func Today() time.Time {
	now := time.Now()
	return Date(now)
}

// Date 截断 Time 中的时间，仅保留日期
func Date(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0,
		0,
		0,
		0,
		t.Location(),
	)
}
