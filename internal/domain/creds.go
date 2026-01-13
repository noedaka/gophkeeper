package domain

type Creds struct {
	ID int

	Login       string
	Password    string
	ServiceName string
	Metadata    string

	UserID int
}
