package domain

import "encoding/json"

type Text struct {
	ID int `json:"-"`

	Text     string `json:"text"`
	Metadata string `json:"metadata"`

	UserID int `json:"-"`
}

func (t *Text) MarshalPlain() ([]byte, error) {
	return json.Marshal(struct {
		Text     string `json:"text"`
		Metadata string `json:"metadata"`
	}{
		Text:     t.Text,
		Metadata: t.Metadata,
	})
}

func UnmarshalPlainText(data []byte) (*Text, error) {
	var t Text
	err := json.Unmarshal(data, &t)
	return &t, err
}
