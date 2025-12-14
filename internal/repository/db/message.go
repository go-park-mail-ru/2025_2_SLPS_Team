package db

import (
	"context"
	"database/sql"
	"fmt"
	"project/domain"
	"strings"
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
		
		// Загружаем вложения для каждого сообщения
		attachments, err := store.GetMessageAttachments(ctx, msg.ID)
		if err != nil {
			dblogger.Error("Failed to get message attachments", zap.Error(err), zap.Int32("messageID", msg.ID))
			// Продолжаем без вложений
		} else {
			msg.Attachments = attachments
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
	
	// Начинаем транзакцию
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		dblogger.Error("Failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			dblogger.Error("Transaction rolled back", zap.Error(err))
		}
	}()

	var messageID int32
	query := `INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`
	dblogger = dblogger.With(zap.Int32("chatID", message.ChatID), zap.String("query", query))
	err = tx.QueryRowContext(ctx, query, message.AuthorID, message.ChatID, message.Text).Scan(&messageID)
	if err != nil {
		dblogger.Error("Failed to create message", zap.Error(err))
		return 0, fmt.Errorf("failed to create message: %w", err)
	}

	// Сохраняем вложения, если они есть
	if len(message.Attachments) > 0 {
		if err := store.saveMessageAttachmentsTx(ctx, tx, messageID, message.Attachments); err != nil {
			dblogger.Error("Failed to save message attachments", zap.Error(err))
			return 0, fmt.Errorf("failed to save message attachments: %w", err)
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit transaction", zap.Error(err))
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	dblogger.Info("Message Created", zap.Int32("messageID", messageID))
	return messageID, nil
}

// GetMessageAttachments возвращает вложения сообщения
func (store *DBMessageStore) GetMessageAttachments(ctx context.Context, messageID int32) ([]string, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "messageStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetMessageAttachments", zap.Int32("messageID", messageID))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`

	dblogger = dblogger.With(zap.String("query", query))
	rows, err := store.db.QueryContext(ctx, query, messageID)
	if err != nil {
		dblogger.Error("Failed to query message attachments", zap.Error(err))
		return nil, fmt.Errorf("failed to query message attachments: %w", err)
	}
	defer rows.Close()

	attachments := []string{}
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			dblogger.Error("Failed to scan attachment", zap.Error(err))
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, filePath)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Rows iteration error", zap.Error(err))
		return nil, err
	}

	dblogger.Info("Message attachments retrieved successfully", zap.Int("count", len(attachments)))
	return attachments, nil
}

// SaveMessageAttachments сохраняет вложения сообщения
func (store *DBMessageStore) SaveMessageAttachments(ctx context.Context, messageID int32, attachments []string) error {
	if len(attachments) == 0 {
		return nil
	}

	start := time.Now()
	dblogger := domain.DBLogger(ctx, "messageStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start SaveMessageAttachments", zap.Int32("messageID", messageID), zap.Int("attachmentsCount", len(attachments)))

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		dblogger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, attachment := range attachments {
		if strings.TrimSpace(attachment) == "" {
			return fmt.Errorf("attachment path cannot be empty")
		}
		if len(attachment) < 1 || len(attachment) > 512 {
			return fmt.Errorf("attachment path length must be between 1 and 512 characters")
		}

		query := `
			INSERT INTO attachments (obj_id, obj_type, file_path)
			VALUES ($1, 'message', $2)
		`
		_, err := tx.ExecContext(ctx, query, messageID, attachment)
		if err != nil {
			tx.Rollback()
			dblogger.Error("Failed to save message attachment", zap.Error(err))
			return fmt.Errorf("failed to save message attachment: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		dblogger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dblogger.Info("Message attachments saved successfully")
	return nil
}

// saveMessageAttachmentsTx сохраняет вложения сообщения в транзакции
func (store *DBMessageStore) saveMessageAttachmentsTx(ctx context.Context, tx *sql.Tx, messageID int32, attachments []string) error {
	if len(attachments) == 0 {
		return nil
	}

	for _, attachment := range attachments {
		if strings.TrimSpace(attachment) == "" {
			return fmt.Errorf("attachment path cannot be empty")
		}
		if len(attachment) < 1 || len(attachment) > 512 {
			return fmt.Errorf("attachment path length must be between 1 and 512 characters")
		}

		query := `
			INSERT INTO attachments (obj_id, obj_type, file_path)
			VALUES ($1, 'message', $2)
		`
		_, err := tx.ExecContext(ctx, query, messageID, attachment)
		if err != nil {
			return fmt.Errorf("failed to save message attachment: %w", err)
		}
	}

	return nil
}