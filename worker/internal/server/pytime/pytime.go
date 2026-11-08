package pytime

import (
	"fmt"
	"strings"
	"time"
)

func Now() time.Time {
	return time.Now().UTC()
}

func Isoformat(moment time.Time) string {
	moment = moment.UTC()
	base := moment.Format("2006-01-02T15:04:05")
	if microseconds := moment.Nanosecond() / 1000; microseconds != 0 {
		base += fmt.Sprintf(".%06d", microseconds)
	}
	return base + "+00:00"
}

func IsoformatOrNil(moment *time.Time) any {
	if moment == nil {
		return nil
	}
	return Isoformat(*moment)
}

var isoLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func ParseISO(text string) (time.Time, bool) {
	normalized := strings.Replace(text, "Z", "+00:00", 1)
	for _, layout := range isoLayouts {
		if parsed, err := time.Parse(layout, normalized); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}
