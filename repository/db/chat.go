package db

import (
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
)

type DBChatStore struct {
	db *sql.DB
}

func NewDBChatStore(db *sql.DB) domain.ChatStore {
	return &DBChatStore{db: db}
}
func (store *DBChatStore) GetOrCreateChatWithUser(selfUserID int, userID int) (int, error) {
	// Упорядочим ID, чтобы избежать дубликатов чатов типа (1,2) и (2,1)
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
	if err == nil {
		tx.Commit()
		return chatID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to query chat: %w", err)
	}

	createChat := `INSERT INTO chats DEFAULT VALUES RETURNING id`
	err = tx.QueryRow(createChat).Scan(&chatID)
	if err != nil {
		return 0, fmt.Errorf("failed to create chat: %w", err)
	}

	insertMembers := `INSERT INTO chat_members (chat_id, member_id) VALUES ($1, $2), ($1, $3)`
	_, err = tx.Exec(insertMembers, chatID, ids[0], ids[1])
	if err != nil {
		return 0, fmt.Errorf("failed to add members: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit tx: %w", err)
	}

	return chatID, nil
}

func (store *DBChatStore) IsMemberOfChat(userID int, chatID int) (bool, error) {
	var success bool
	query := `SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 and member_id = $2)`
	err := store.db.QueryRow(query, chatID, userID).Scan(&success)
	if err != nil {
		return false, err
	}

	return success, nil
}

func (store *DBChatStore) IsChatExist(chatID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1 )`
	err := store.db.QueryRow(query, chatID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
