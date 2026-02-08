package dto

type MediaPhotoResponse struct {
	ID       int64  `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type MediaPhotosListResponse struct {
	Items []MediaPhotoResponse `json:"items"`
}
