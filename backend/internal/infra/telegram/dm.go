package telegram

func BuildDMDeepLink(username string) string {
	if username == "" {
		return ""
	}
	return "https://t.me/" + username
}
