package model

type PartnerOffer struct {
	ID          int64  `json:"id"`
	PartnerName string `json:"partner_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CTAURL      string `json:"cta_url"`
}
