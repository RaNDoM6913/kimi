package dto

type TelegramAuthRequest struct {
	InitData string `json:"init_data"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthMeResponse struct {
	ID   int64  `json:"id"`
	Role string `json:"role"`
}

type AuthTokensResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresInSec int64          `json:"expires_in_sec"`
	Me           AuthMeResponse `json:"me"`
}

type LogoutResponse struct {
	OK bool `json:"ok"`
}
