CREATE TABLE records (
	id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,  
    metadata    TEXT NOT NULL DEFAULT '',
    record_type TEXT NOT NULL,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE
);