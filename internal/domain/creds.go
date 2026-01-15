package domain

import "encoding/json"

type Creds struct {
	ID int `json:"-"`

	Login       string `json:"login"`
	Password    string `json:"password"`
	ServiceName string `json:"service_name"`
	Metadata    string `json:"metadata"`

	UserID int `json:"-"`
}

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

func UnmarshalPlainCreds(data []byte) (*Creds, error) {
	var c Creds
	err := json.Unmarshal(data, &c)
	return &c, err
}
