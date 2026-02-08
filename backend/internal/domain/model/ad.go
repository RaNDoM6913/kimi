package model

type Ad struct {
	ID       int64  `json:"id"`
	Partner  string `json:"partner,omitempty"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	AssetURL string `json:"asset_url"`
	ClickURL string `json:"click_url"`
	IsActive bool   `json:"is_active"`
}
