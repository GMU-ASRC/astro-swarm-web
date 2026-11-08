package evaluations

import (
	"time"

	"astroswarm/worker/internal/server/pytime"
)

type timeValue struct {
	moment time.Time
}

func newTimeValue(moment *time.Time) *timeValue {
	if moment == nil {
		return nil
	}
	return &timeValue{moment: *moment}
}

func (value *timeValue) before(other *timeValue) bool {
	if value == nil {
		return false
	}
	if other == nil {
		return true
	}
	return value.moment.Before(other.moment)
}

func (value *timeValue) isoformat() any {
	if value == nil {
		return nil
	}
	return pytime.Isoformat(value.moment)
}
