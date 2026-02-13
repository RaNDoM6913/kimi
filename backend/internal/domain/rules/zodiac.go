package rules

import "time"

// ZodiacFromBirthdate maps Gregorian month/day to western zodiac sign.
// Boundaries:
// aries: Mar 21 - Apr 19
// taurus: Apr 20 - May 20
// gemini: May 21 - Jun 20
// cancer: Jun 21 - Jul 22
// leo: Jul 23 - Aug 22
// virgo: Aug 23 - Sep 22
// libra: Sep 23 - Oct 22
// scorpio: Oct 23 - Nov 21
// sagittarius: Nov 22 - Dec 21
// capricorn: Dec 22 - Jan 19
// aquarius: Jan 20 - Feb 18
// pisces: Feb 19 - Mar 20
func ZodiacFromBirthdate(d time.Time) string {
	if d.IsZero() {
		return ""
	}

	m := d.UTC().Month()
	day := d.UTC().Day()

	switch m {
	case time.March:
		if day >= 21 {
			return "aries"
		}
		return "pisces"
	case time.April:
		if day >= 20 {
			return "taurus"
		}
		return "aries"
	case time.May:
		if day >= 21 {
			return "gemini"
		}
		return "taurus"
	case time.June:
		if day >= 21 {
			return "cancer"
		}
		return "gemini"
	case time.July:
		if day >= 23 {
			return "leo"
		}
		return "cancer"
	case time.August:
		if day >= 23 {
			return "virgo"
		}
		return "leo"
	case time.September:
		if day >= 23 {
			return "libra"
		}
		return "virgo"
	case time.October:
		if day >= 23 {
			return "scorpio"
		}
		return "libra"
	case time.November:
		if day >= 22 {
			return "sagittarius"
		}
		return "scorpio"
	case time.December:
		if day >= 22 {
			return "capricorn"
		}
		return "sagittarius"
	case time.January:
		if day >= 20 {
			return "aquarius"
		}
		return "capricorn"
	case time.February:
		if day >= 19 {
			return "pisces"
		}
		return "aquarius"
	default:
		return ""
	}
}
