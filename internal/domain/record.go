package domain

// Record структура для зашифрованных записей. Соответствует таблице records в БД
type Record struct {
	ID         int    `db:"id"`
	Ciphertext []byte `db:"ciphertext"`
	Nonce      []byte `db:"nonce"`
	Metadata   string `db:"metadata"`
	RecordType string `db:"record_type"`
	UserID     string `db:"user_id"`
}

// RecordInfo упрощённая структура для списка записей. Возвращается в List-операциях
type RecordInfo struct {
	ID         int    `db:"id"`
	Metadata   string `db:"metadata"`
	RecordType string `db:"record_type"`
}
