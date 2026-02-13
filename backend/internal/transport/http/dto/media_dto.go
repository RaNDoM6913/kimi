package dto

type MediaPhotoResponse struct {
	ID       int64  `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type MediaPhotosListResponse struct {
	Items []MediaPhotoResponse `json:"items"`
}

type CandidatePhotoResponse struct {
	Slot int    `json:"slot"`
	URL  string `json:"url"`
	W    *int   `json:"w"`
	H    *int   `json:"h"`
}

type CandidatePhotosResponse struct {
	UserID int64                    `json:"user_id"`
	Photos []CandidatePhotoResponse `json:"photos"`
}
