package adminhttp

import (
	"errors"
	"net/url"
	"strconv"
)

func shouldFallback(dual bool, err error) bool {
	if err == nil {
		return false
	}
	return dual && IsFallbackable(err)
}

func shouldFallbackWithNotFound(dual bool, err error) bool {
	if err == nil || !dual {
		return false
	}
	if IsFallbackable(err) {
		return true
	}

	var reqErr *RequestError
	if errors.As(err, &reqErr) && reqErr.StatusCode == 404 {
		return true
	}
	return false
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}
