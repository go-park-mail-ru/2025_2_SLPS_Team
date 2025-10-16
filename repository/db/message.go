package db

import (
	"database/sql"
	"project/domain"
)

type DBMessageStore struct {
	db *sql.DB
}

func NewDBMessageStore(db *sql.DB) domain.MessageStore {
	return &DBMessageStore{db: db}
}
func (store *DBMessageStore) GetMessagesByChatId(chatID int, limit int, offset int) ([]domain.Message, error) {
	var messages []domain.Message

	query := `SELECT id, author_id, chat_id, text, created_at

              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`

	rows, err := store.db.Query(query, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.AuthorID,
			&msg.ChatID,
			&msg.Text,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (store *DBMessageStore) CreateMessage(message domain.Message) (int, error) {

	var messageID int
	query := `INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`
	err := store.db.QueryRow(query, message.AuthorID, message.ChatID, message.Text).Scan(&messageID)
	if err != nil {
		return 0, err
	}
	return messageID, nil
}
