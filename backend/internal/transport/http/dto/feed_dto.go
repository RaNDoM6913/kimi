package dto

type FeedAdCardResponse struct {
	ID       int64  `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title,omitempty"`
	AssetURL string `json:"asset_url"`
	ClickURL string `json:"click_url"`
}

type FeedItemResponse struct {
	IsAd        bool                `json:"is_ad"`
	Ad          *FeedAdCardResponse `json:"ad,omitempty"`
	UserID      int64               `json:"user_id,omitempty"`
	DisplayName string              `json:"display_name,omitempty"`
	Age         int                 `json:"age,omitempty"`
	CityID      string              `json:"city_id,omitempty"`
	City        string              `json:"city,omitempty"`
	DistanceKM  *float64            `json:"distance_km,omitempty"`
}

type FeedResponse struct {
	Items      []FeedItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}
