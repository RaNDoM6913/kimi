package dto

type ProfileLocationRequest struct {
	Lat *float64 `json:"lat"`
	Lon *float64 `json:"lon"`
}

type ProfileLocationResponse struct {
	CityID   string `json:"city_id"`
	CityName string `json:"city_name"`
}
