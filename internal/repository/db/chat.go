package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"project/domain"
	"time"

	"github.com/lib/pq"
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
	dblogger := domain.DBLogger(ctx, "chatStore")
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
	row := tx.QueryRow(query, pq.Array(ids))
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
		dblogger.Error("Failed to add members", zap.String("query", insertMembers), zap.Error(err))
		return 0, fmt.Errorf("failed to add members: %w", err)
	}

	if err = tx.Commit(); err != nil {
		dblogger.Error("Failed to commit tx", zap.Error(err))
		return 0, fmt.Errorf("failed to commit tx: %w", err)
	}
	dblogger.Info("Chat created and return", zap.Int("chatID", chatID))
	return chatID, nil
}

func (store *DBChatStore) IsMemberOfChat(ctx context.Context, userID int, chatID int) (bool, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
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
	dblogger := domain.DBLogger(ctx, "chatStore")
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

// реализовать пагинацию чатов и сообщений через id или время последнего объекта в будущем
func (store *DBChatStore) GetUserFullChats(ctx context.Context, userID int, limit, offset int) ([]domain.FullChat, error) {

	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserFullChats")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `

WITH last_messages AS (
    SELECT DISTINCT ON (chat_id)
        id AS message_id,
        chat_id,
        author_id,
        text,
        created_at
    FROM messages
    ORDER BY chat_id, created_at DESC
)
SELECT
    c.id AS chat_id,
    c.is_group,
    COALESCE(private_user.chat_name, c.name) AS chat_name,
    COALESCE(private_user.chat_avatar, c.avatar) AS chat_avatar,
    
    lm.message_id AS last_message_id,
    lm.text AS last_message_text,
    lm.created_at AS last_message_created_at,
    
    author_user.id AS last_message_author_id,
    author_profile.first_name || ' ' || author_profile.last_name AS last_message_author_name,
    author_profile.avatar_path AS last_message_author_avatar

FROM chat_members cm
JOIN chats c ON c.id = cm.chat_id
JOIN last_messages lm ON lm.chat_id = c.id
JOIN users author_user ON author_user.id = lm.author_id
JOIN profiles author_profile ON author_profile.user_id = author_user.id

LEFT JOIN LATERAL (
    SELECT 
        p.first_name || ' ' || p.last_name AS chat_name,
        p.avatar_path AS chat_avatar
    FROM chat_members cm2
    JOIN profiles p ON p.user_id = cm2.member_id
    WHERE cm2.chat_id = c.id AND cm2.member_id != $1
    LIMIT 1
) private_user ON NOT c.is_group

WHERE cm.member_id = $1
ORDER BY lm.created_at DESC
LIMIT $2 OFFSET $3;

    `

	rows, err := store.db.Query(query, userID, limit, offset)
	dblogger = dblogger.With(zap.String("query", query))
	if err != nil {
		dblogger.Error("Failed to find chats by user", zap.Error(err), zap.Int("limit", limit), zap.Int("offset", offset))
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	chats := []domain.FullChat{}
	for rows.Next() {
		var c domain.FullChat
		var m domain.Message
		var u domain.ShortProfile
		err := rows.Scan(
			&c.ID,
			&c.IsGroup,
			&c.Name,
			&c.AvatarPath,
			&m.ID,
			&m.Text,
			&m.CreatedAt,
			&u.UserID,
			&u.FullName,
			&u.AvatarPath,
		)
		m.AuthorID = u.UserID
		m.ChatID = c.ID
		c.LastMessage = m
		c.LastMessageAuthor = u
		if err != nil {
			dblogger.Error("Failed to read chat rows", zap.Error(err))
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		chats = append(chats, c)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Failed to read chat rows", zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}

	dblogger.Info("Chats returns")
	return chats, nil
}

func (store *DBChatStore) GetOtherChatMembersIdByAuthorId(ctx context.Context, userID int, chatID int) ([]int, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetOtherChatMembersIdByAuthorId")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `
	SELECT member_id
	FROM chat_members
	WHERE member_id != $1 and chat_id = $2
	`
	rows, err := store.db.Query(query, userID, chatID)
	dblogger = dblogger.With(zap.String("query", query), zap.Int("userID", userID), zap.Int("ChatID", chatID))
	if err != nil {
		dblogger.Error("Failed to find members of chat")
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	memberIDs := []int{}
	for rows.Next() {
		var memberID int
		err := rows.Scan(
			&memberID,
		)
		if err != nil {
			dblogger.Error("Failed to read member rows", zap.Error(err))
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		memberIDs = append(memberIDs, memberID)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Failed to read member rows", zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}

	dblogger.Info("Members returns")
	return memberIDs, nil
}

func (store *DBChatStore) GetFullChatByIDAndSenderID(ctx context.Context, userID int, chatID int) (*domain.FullChat, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserFullChats")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `

WITH last_message AS (
    SELECT DISTINCT ON (chat_id)
        id AS message_id,
        chat_id,
        author_id,
        text,
        created_at
    FROM messages
    WHERE chat_id = $1
    ORDER BY chat_id, created_at DESC
)
SELECT
    c.id AS chat_id,
    c.is_group,
    COALESCE(private_user.chat_name, c.name) AS chat_name,
    COALESCE(private_user.chat_avatar, c.avatar) AS chat_avatar,

    lm.message_id AS last_message_id,
    lm.text AS last_message_text,
    lm.created_at AS last_message_created_at,

    author_user.id AS last_message_author_id,
    author_profile.first_name || ' ' || author_profile.last_name AS last_message_author_name,
    author_profile.avatar_path AS last_message_author_avatar

FROM chats c
LEFT JOIN last_message lm ON lm.chat_id = c.id
LEFT JOIN users author_user ON author_user.id = lm.author_id
LEFT JOIN profiles author_profile ON author_profile.user_id = author_user.id

LEFT JOIN LATERAL (
    SELECT 
        p.first_name || ' ' || p.last_name AS chat_name,
        p.avatar_path AS chat_avatar
    FROM chat_members cm2
    JOIN profiles p ON p.user_id = cm2.member_id
    WHERE cm2.chat_id = c.id AND cm2.member_id = $2
    LIMIT 1
) private_user ON NOT c.is_group

WHERE c.id = $1;
    `

	dblogger = dblogger.With(zap.String("query", query))

	row := store.db.QueryRow(query, chatID, userID)

	var c domain.FullChat
	var m domain.Message
	var u domain.ShortProfile

	err := row.Scan(
		&c.ID,
		&c.IsGroup,
		&c.Name,
		&c.AvatarPath,
		&m.ID,
		&m.Text,
		&m.CreatedAt,
		&u.UserID,
		&u.FullName,
		&u.AvatarPath,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("chat not found: %w", err)
		}
		dblogger.Error("Failed to scan chat row", zap.Error(err))
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	m.AuthorID = u.UserID
	m.ChatID = c.ID
	c.LastMessage = m
	c.LastMessageAuthor = u

	dblogger.Info("Chat returned successfully", zap.Int("chat_id", chatID))
	return &c, nil
}
