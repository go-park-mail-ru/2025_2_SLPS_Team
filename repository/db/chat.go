package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"project/internal/service"
	"time"

	"go.uber.org/zap"
)

type DBChatStore struct {
	db *sql.DB
}

func NewDBChatStore(db *sql.DB) domain.ChatStore {
	return &DBChatStore{db: db}
}
func (store *DBChatStore) GetOrCreateChatWithUser(ctx context.Context, selfUserID int, userID int) (int, error) {

	start := time.Now()
	dblogger := service.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetOrCreateChatWithUser")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	ids := []int{selfUserID, userID}
	if selfUserID > userID {
		ids[0], ids[1] = userID, selfUserID
	}
	var chatID int
	tx, err := store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
        SELECT chat_id
        FROM chat_members
        WHERE member_id = ANY($1)
        GROUP BY chat_id
        HAVING COUNT(*) = 2 AND bool_and(member_id = ANY($1))
        LIMIT 1
    `
	row := tx.QueryRow(query, ids)
	err = row.Scan(&chatID)
	dblogger = dblogger.With(zap.Int("userID", userID))
	if err == nil {
		tx.Commit()
		dblogger.Info("Chat is exist and return", zap.String("query", query), zap.Int("chatID", chatID))
		return chatID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		dblogger.Error("Failed to get chat", zap.String("query", query), zap.Error(err))
		return 0, fmt.Errorf("failed to query chat: %w", err)
	}

	createChat := `INSERT INTO chats DEFAULT VALUES RETURNING id`
	err = tx.QueryRow(createChat).Scan(&chatID)
	if err != nil {
		dblogger.Error("Failed to create chat", zap.String("query", createChat), zap.Error(err))
		return 0, fmt.Errorf("failed to create chat: %w", err)
	}

	insertMembers := `INSERT INTO chat_members (chat_id, member_id) VALUES ($1, $2), ($1, $3)`
	_, err = tx.Exec(insertMembers, chatID, ids[0], ids[1])
	if err != nil {
		dblogger.Error("failed to add members", zap.String("query", insertMembers), zap.Error(err))
		return 0, fmt.Errorf("failed to add members: %w", err)
	}

	if err = tx.Commit(); err != nil {
		dblogger.Error("failed to commit tx", zap.Error(err))
		return 0, fmt.Errorf("failed to commit tx: %w", err)
	}
	dblogger.Info("Chat created and return", zap.Int("chatID", chatID))
	return chatID, nil
}

func (store *DBChatStore) IsMemberOfChat(ctx context.Context, userID int, chatID int) (bool, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start IsMemberOfChat")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	var success bool
	query := `SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 and member_id = $2)`
	dblogger = dblogger.With(zap.Int("userID", userID), zap.Int("chatID", chatID), zap.String("query", query))
	err := store.db.QueryRow(query, chatID, userID).Scan(&success)
	if err != nil {
		dblogger.Error("failed to find member", zap.Error(err))
		return false, err
	}

	dblogger.Info("Member find successfully")
	return success, nil
}

func (store *DBChatStore) IsChatExist(ctx context.Context, chatID int) (bool, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start IsChatExist")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1 )`
	dblogger = dblogger.With(zap.Int("chatID", chatID), zap.String("query", query))
	err := store.db.QueryRow(query, chatID).Scan(&exists)
	if err != nil {
		dblogger.Error("failed to find chat", zap.Error(err))
		return false, err
	}

	dblogger.Info("Chat find successfully")
	return exists, nil
}
