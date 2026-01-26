package domain

import "encoding/json"

// Creds определяет структуру логина/пароля и данных о сервисе
type Creds struct {
	ID int `json:"-"`

	Login       string `json:"login"`
	Password    string `json:"password"`
	ServiceName string `json:"service_name"`
	Metadata    string `json:"metadata"`

	UserID int `json:"-"`
}

// MarshalPlain сериализует только чувствительные поля
func (c *Creds) MarshalPlain() ([]byte, error) {
	return json.Marshal(struct {
		Login       string `json:"login"`
		Password    string `json:"password"`
		ServiceName string `json:"service_name"`
		Metadata    string `json:"metadata"`
	}{
		Login:       c.Login,
		Password:    c.Password,
		ServiceName: c.ServiceName,
		Metadata:    c.Metadata,
	})
}

// UnmarshalPlainCreds заполняет структуру из plaintext json
func UnmarshalPlainCreds(data []byte) (*Creds, error) {
	var c Creds
	err := json.Unmarshal(data, &c)
	return &c, err
}
