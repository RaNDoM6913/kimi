package dto

import "encoding/json"

type FeedAdCardResponse struct {
	ID       int64  `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title,omitempty"`
	AssetURL string `json:"asset_url"`
	ClickURL string `json:"click_url"`
}

type FeedItemResponse struct {
	IsAd            bool                `json:"is_ad"`
	Ad              *FeedAdCardResponse `json:"ad,omitempty"`
	UserID          int64               `json:"user_id,omitempty"`
	DisplayName     string              `json:"display_name,omitempty"`
	Age             int                 `json:"age,omitempty"`
	Zodiac          string              `json:"zodiac,omitempty"`
	PrimaryGoal     string              `json:"primary_goal,omitempty"`
	PrimaryPhotoURL *NullableString     `json:"primary_photo_url,omitempty"`
	CityID          string              `json:"city_id,omitempty"`
	City            string              `json:"city,omitempty"`
	DistanceKM      *float64            `json:"distance_km,omitempty"`
}

type FeedResponse struct {
	Items      []FeedItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type NullableString struct {
	Value *string
}

func (n NullableString) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	if n == nil {
		return nil
	}
	if string(data) == "null" {
		n.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}
