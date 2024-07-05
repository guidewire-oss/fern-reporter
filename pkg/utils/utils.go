package utils

import "time"

const DateLayoutFormat = "2006-01-02 15:04:05"

func CalculateDuration(start, end time.Time) string {
	duration := end.Sub(start)
	return duration.String() // or format as needed
}

func FormatDate(t time.Time) string {
	return t.Format(DateLayoutFormat)
}
