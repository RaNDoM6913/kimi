package payments

type StarsProvider struct {
	Name string
}

func NewStarsProvider() *StarsProvider {
	return &StarsProvider{Name: "telegram_stars"}
}
