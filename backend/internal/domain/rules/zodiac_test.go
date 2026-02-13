package rules

import (
	"testing"
	"time"
)

func TestZodiacFromBirthdateBoundaries(t *testing.T) {
	cases := []struct {
		name string
		date time.Time
		want string
	}{
		{name: "aries_start", date: time.Date(1990, time.March, 21, 0, 0, 0, 0, time.UTC), want: "aries"},
		{name: "aries_end", date: time.Date(1990, time.April, 19, 0, 0, 0, 0, time.UTC), want: "aries"},
		{name: "taurus_start", date: time.Date(1990, time.April, 20, 0, 0, 0, 0, time.UTC), want: "taurus"},
		{name: "gemini_start", date: time.Date(1990, time.May, 21, 0, 0, 0, 0, time.UTC), want: "gemini"},
		{name: "cancer_start", date: time.Date(1990, time.June, 21, 0, 0, 0, 0, time.UTC), want: "cancer"},
		{name: "leo_start", date: time.Date(1990, time.July, 23, 0, 0, 0, 0, time.UTC), want: "leo"},
		{name: "virgo_start", date: time.Date(1990, time.August, 23, 0, 0, 0, 0, time.UTC), want: "virgo"},
		{name: "libra_start", date: time.Date(1990, time.September, 23, 0, 0, 0, 0, time.UTC), want: "libra"},
		{name: "scorpio_start", date: time.Date(1990, time.October, 23, 0, 0, 0, 0, time.UTC), want: "scorpio"},
		{name: "sagittarius_start", date: time.Date(1990, time.November, 22, 0, 0, 0, 0, time.UTC), want: "sagittarius"},
		{name: "capricorn_start", date: time.Date(1990, time.December, 22, 0, 0, 0, 0, time.UTC), want: "capricorn"},
		{name: "aquarius_start", date: time.Date(1990, time.January, 20, 0, 0, 0, 0, time.UTC), want: "aquarius"},
		{name: "pisces_start", date: time.Date(1990, time.February, 19, 0, 0, 0, 0, time.UTC), want: "pisces"},
		{name: "pisces_end", date: time.Date(1990, time.March, 20, 0, 0, 0, 0, time.UTC), want: "pisces"},
		{name: "zero", date: time.Time{}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ZodiacFromBirthdate(tc.date)
			if got != tc.want {
				t.Fatalf("unexpected zodiac: got %q want %q", got, tc.want)
			}
		})
	}
}
