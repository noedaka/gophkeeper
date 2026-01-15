package domain

import "encoding/json"

type Card struct {
	ID int `json:"-"`

	CardNumber     string `json:"card_number"`
	CardHolderName string `json:"card_holder_name"`
	ExpiryDate     string `json:"expiry_date"`
	CVV            string `json:"cvv"`
	Metadata       string `json:"metadata"`

	UserID int `json:"-"`
}

// MarshalPlain — сериализует только чувствительные поля
func (c *Card) MarshalPlain() ([]byte, error) {
	return json.Marshal(struct {
		CardNumber     string `json:"card_number"`
		CardHolderName string `json:"card_holder_name"`
		ExpiryDate     string `json:"expiry_date"`
		CVV            string `json:"cvv"`
		Metadata       string `json:"metadata"`
	}{
		CardNumber:     c.CardNumber,
		CardHolderName: c.CardHolderName,
		ExpiryDate:     c.ExpiryDate,
		CVV:            c.CVV,
		Metadata:       c.Metadata,
	})
}

// UnmarshalPlain — заполняет структуру из plaintext json
func UnmarshalPlainCard(data []byte) (*Card, error) {
	var c Card
	err := json.Unmarshal(data, &c)
	return &c, err
}
