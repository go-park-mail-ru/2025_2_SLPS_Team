package db

import (
	"context"
	"database/sql"
	"project/domain"
	"time"

	"go.uber.org/zap"
)

type DBMessageStore struct {
	db *sql.DB
}

func NewDBMessageStore(db *sql.DB) domain.MessageStore {
	return &DBMessageStore{db: db}
}
func (store *DBMessageStore) GetMessagesByChatId(ctx context.Context, chatID int32, limit int32, offset int32) ([]domain.Message, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "messageStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetMessagesByChatId")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	var messages []domain.Message

	query := `SELECT id, author_id, chat_id, text, created_at

              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`
	dblogger = dblogger.With(zap.Int32("chatID", chatID), zap.String("query", query))
	rows, err := store.db.Query(query, chatID, limit, offset)
	if err != nil {
		dblogger.Error("Failed to find messages by chat", zap.Error(err))
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
			dblogger.Error("Failed to read message rows", zap.Error(err))
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Failed to read message rows", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Messages return")
	return messages, nil
}

func (store *DBMessageStore) CreateMessage(ctx context.Context, message domain.Message) (int32, error) {

	start := time.Now()
	dblogger := domain.DBLogger(ctx, "messageStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start CreateMessage")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	var messageID int32
	query := `INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`
	dblogger = dblogger.With(zap.Int32("chatID", message.ChatID), zap.String("query", query))
	err := store.db.QueryRow(query, message.AuthorID, message.ChatID, message.Text).Scan(&messageID)
	if err != nil {
		dblogger.Error("Failed to create message", zap.Error(err))
		return 0, err
	}

	dblogger.Info("Message Created", zap.Int32("messageID", messageID))
	return messageID, nil
}
