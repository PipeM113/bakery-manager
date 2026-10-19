// Package clock is the one place that knows what time it is for the business: the bakery lives
// in Chile, so "today" and "this month" are the ones of America/Santiago, whatever the server's
// own clock says (Vercel runs in UTC).
package clock

import "time"

// Santiago is the business time zone.
var Santiago = loadSantiago()

func loadSantiago() *time.Location {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		return time.UTC
	}
	return loc
}

// Clock tells the current instant. Tests pass a fixed one.
type Clock func() time.Time

// System is the real clock.
var System Clock = time.Now

// Fixed is a clock that always says t.
func Fixed(t time.Time) Clock { return func() time.Time { return t } }

// Today is the business date, as YYYY-MM-DD.
func (c Clock) Today() string { return c().In(Santiago).Format("2006-01-02") }

// Month is the business month and year.
func (c Clock) Month() (month, year int) {
	t := c().In(Santiago)
	return int(t.Month()), t.Year()
}

// LastMonths lists the n months that end with the business month of now, oldest first, as
// "YYYY-MM". (Sprint 5 red stub: not implemented yet.)
func LastMonths(now time.Time, n int) []string { return nil }
