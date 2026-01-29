package domain

// BinaryRecord определяет структуру бинарной записи
type BinaryRecord struct {
	ID       int    `db:"id"`
	UserID   string `db:"user_id"`
	S3Key    string `db:"s3_key"`
	Metadata string `db:"metadata"`
}
