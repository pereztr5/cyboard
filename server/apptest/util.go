package apptest

import "time"

func MustParseTime(timestr string) time.Time {
	// Expects format: "2018-07-29T09:15:00.000-04:00"
	ts, err := time.Parse(time.RFC3339, timestr)
	if err != nil {
		panic(err)
	}
	return ts
}
