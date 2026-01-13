package domain

type Card struct {
	ID int

	CardNumber     string
	CardHolderName string
	ExpiryDate     string
	CVV            string
	Metadata       string

	UserID int
}
