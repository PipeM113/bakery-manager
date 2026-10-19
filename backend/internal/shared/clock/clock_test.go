package clock

import (
	"slices"
	"testing"
	"time"
)

// Sprint 5, rules T1 and T4. The instants are written in UTC; the expected dates are the ones a
// person standing in Santiago would read on the calendar. Chile moves its clocks at 24:00 on the
// first Saturday of April (back, to UTC-4) and of September (forward, to UTC-3).

func utc(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestT1_TodayIsTheSantiagoDate(t *testing.T) {
	cases := []struct{ name, instant, want string }{
		{"UTC is already the next day, Santiago is not (summer, UTC-3)", "2026-04-01T02:30:00Z", "2026-03-31"},
		{"same in winter (UTC-4)", "2026-07-01T03:30:00Z", "2026-06-30"},
		{"midday", "2026-03-31T15:00:00Z", "2026-03-31"},
		{"just after Santiago midnight in summer", "2026-03-31T03:00:00Z", "2026-03-31"},
		{"one minute before it", "2026-03-31T02:59:00Z", "2026-03-30"},
		{"clocks go back: 23:30 at UTC-3", "2026-04-05T02:30:00Z", "2026-04-04"},
		{"clocks go back: 23:30 again at UTC-4", "2026-04-05T03:30:00Z", "2026-04-04"},
		{"clocks go back: the new day", "2026-04-05T04:30:00Z", "2026-04-05"},
		{"clocks go forward: 23:30 at UTC-4", "2026-09-06T03:30:00Z", "2026-09-05"},
		{"clocks go forward: 01:30 at UTC-3", "2026-09-06T04:30:00Z", "2026-09-06"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Fixed(utc(c.instant)).Today(); got != c.want {
				t.Errorf("Today() at %s = %s, want %s", c.instant, got, c.want)
			}
		})
	}
}

func TestT1_MonthIsTheSantiagoMonth(t *testing.T) {
	month, year := Fixed(utc("2026-04-01T02:30:00Z")).Month()
	if month != 3 || year != 2026 {
		t.Errorf("Month() = %d/%d, want 3/2026", month, year)
	}
	month, year = Fixed(utc("2027-01-01T02:30:00Z")).Month()
	if month != 12 || year != 2026 {
		t.Errorf("Month() = %d/%d, want 12/2026 (the year changes in Santiago three hours later)", month, year)
	}
}

// T4: the months are counted from the first of the month, so the day of the month cannot make
// one disappear or repeat (adding minus one month to the 31st of March lands on the 3rd of March).
func TestT4_LastMonths(t *testing.T) {
	cases := []struct {
		name, instant string
		n             int
		want          []string
	}{
		{"from the 31st of March", "2026-03-31T15:00:00Z", 4, []string{"2025-12", "2026-01", "2026-02", "2026-03"}},
		{"from the 31st of January", "2026-01-31T15:00:00Z", 3, []string{"2025-11", "2025-12", "2026-01"}},
		{"from the 30th of March", "2026-03-30T15:00:00Z", 2, []string{"2026-02", "2026-03"}},
		{"leap year, 29th of February", "2024-02-29T15:00:00Z", 3, []string{"2023-12", "2024-01", "2024-02"}},
		{"leap year, 31st of March", "2024-03-31T15:00:00Z", 3, []string{"2024-01", "2024-02", "2024-03"}},
		{"a year, across the year change", "2026-03-31T15:00:00Z", 12, []string{
			"2025-04", "2025-05", "2025-06", "2025-07", "2025-08", "2025-09",
			"2025-10", "2025-11", "2025-12", "2026-01", "2026-02", "2026-03"}},
		{"one month", "2026-03-31T15:00:00Z", 1, []string{"2026-03"}},
		{"the month is the Santiago one, not the UTC one", "2026-04-01T02:30:00Z", 2, []string{"2026-02", "2026-03"}},
		{"across the clock change", "2026-04-30T15:00:00Z", 3, []string{"2026-02", "2026-03", "2026-04"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := LastMonths(utc(c.instant), c.n)
			if !slices.Equal(got, c.want) {
				t.Errorf("LastMonths(%s, %d) = %v, want %v", c.instant, c.n, got, c.want)
			}
		})
	}
}
