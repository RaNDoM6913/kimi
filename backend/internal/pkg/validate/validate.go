package validate

import "strings"

func Required(value string) bool {
	return strings.TrimSpace(value) != ""
}
