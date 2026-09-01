package page

import "time"

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}
