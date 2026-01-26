package domain

import "encoding/json"

// Text определяет структуру текста
type Text struct {
	ID int `json:"-"`

	Text     string `json:"text"`
	Metadata string `json:"metadata"`

	UserID int `json:"-"`
}

// MarshalPlain сериализует только чувствительные поля
func (t *Text) MarshalPlain() ([]byte, error) {
	return json.Marshal(struct {
		Text     string `json:"text"`
		Metadata string `json:"metadata"`
	}{
		Text:     t.Text,
		Metadata: t.Metadata,
	})
}

// UnmarshalPlainCreds заполняет структуру из plaintext json
func UnmarshalPlainText(data []byte) (*Text, error) {
	var t Text
	err := json.Unmarshal(data, &t)
	return &t, err
}
