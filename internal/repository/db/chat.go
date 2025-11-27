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
func (store *DBChatStore) GetOrCreateChatWithUser(ctx context.Context, selfUserID int32, userID int32) (int32, error) {

	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetOrCreateChatWithUser")
	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	ids := []int32{selfUserID, userID}
	if selfUserID > userID {
		ids[0], ids[1] = userID, selfUserID
	}
	var chatID int32
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
	dblogger = dblogger.With(zap.Int32("userID", userID))
	if err == nil {
		tx.Commit()
		dblogger.Info("Chat is exist and return", zap.String("query", query), zap.Int32("chatID", chatID))
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
	dblogger.Info("Chat created and return", zap.Int32("chatID", chatID))
	return chatID, nil
}

func (store *DBChatStore) IsMemberOfChat(ctx context.Context, userID int32, chatID int32) (bool, error) {
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
	dblogger = dblogger.With(zap.Int32("userID", userID), zap.Int32("chatID", chatID), zap.String("query", query))
	err := store.db.QueryRow(query, chatID, userID).Scan(&success)
	if err != nil {
		dblogger.Error("failed to find member", zap.Error(err))
		return false, err
	}

	dblogger.Info("Member find successfully")
	return success, nil
}

func (store *DBChatStore) IsChatExist(ctx context.Context, chatID int32) (bool, error) {
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
	dblogger = dblogger.With(zap.Int32("chatID", chatID), zap.String("query", query))
	err := store.db.QueryRow(query, chatID).Scan(&exists)
	if err != nil {
		dblogger.Error("failed to find chat", zap.Error(err))
		return false, err
	}

	dblogger.Info("Chat find successfully")
	return exists, nil
}

// реализовать пагинацию чатов и сообщений через id или время последнего объекта в будущем
func (store *DBChatStore) GetUserFullChats(ctx context.Context, userID int32, limit, offset int32) ([]domain.FullChat, []int32, error) {

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
    SELECT DISTINCT ON (m.chat_id)
        m.chat_id,
        m.id AS message_id,
        m.text,
        m.created_at,
        m.author_id
    FROM messages m
    WHERE m.chat_id IN (
        SELECT chat_id
        FROM chat_members
        WHERE member_id = $1
    )
    ORDER BY m.chat_id, m.created_at DESC
),
unread_counts AS (
    SELECT
        cm.chat_id,
        COUNT(m.id) AS unread_count
    FROM chat_members cm
    LEFT JOIN messages m 
        ON m.chat_id = cm.chat_id
        AND m.id > cm.last_read_message_id
    WHERE cm.member_id = $1
    GROUP BY cm.chat_id
),
private_user_ids AS (
    SELECT
        cm2.chat_id,
        cm2.member_id AS private_user_id
    FROM chat_members cm2
    JOIN chat_members self ON self.chat_id = cm2.chat_id
    WHERE self.member_id = $1
      AND cm2.member_id != $1
)
SELECT
    c.id AS chat_id,
    c.is_group,
    c.name AS chat_name,
    c.avatar AS chat_avatar,
    lm.message_id AS last_message_id,
    lm.text AS last_message_text,
    lm.created_at AS last_message_created_at,
    lm.author_id AS last_message_author_id,
    CASE 
        WHEN NOT c.is_group THEN pu.private_user_id
        ELSE NULL
    END AS private_user_id,
    COALESCE(uc.unread_count, 0) AS unread_count,
    cm.last_read_message_id
FROM chat_members cm
JOIN chats c ON c.id = cm.chat_id
JOIN last_messages lm ON lm.chat_id = c.id
JOIN unread_counts uc ON uc.chat_id = c.id
LEFT JOIN private_user_ids pu ON pu.chat_id = c.id
WHERE cm.member_id = $1
ORDER BY lm.created_at DESC
LIMIT $2 OFFSET $3;

    `

	rows, err := store.db.Query(query, userID, limit, offset)
	dblogger = dblogger.With(zap.String("query", query))
	if err != nil {
		dblogger.Error("Failed to find chats by user", zap.Error(err), zap.Int32("limit", limit), zap.Int32("offset", offset))
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	chats := []domain.FullChat{}
	userIDSet := map[int32]struct{}{}
	for rows.Next() {
		var c domain.FullChat
		var m domain.Message
		var lmUserID int32
		var privateUserID *int32
		err := rows.Scan(
			&c.ID,
			&c.IsGroup,
			&c.Name,
			&c.AvatarPath,
			&m.ID,
			&m.Text,
			&m.CreatedAt,
			&lmUserID,
			&privateUserID,
			&c.UnreadCounts,
			&c.LastReadMessageID,
		)
		m.AuthorID = lmUserID
		m.ChatID = c.ID
		c.LastMessage = m
		c.UserIDWith = privateUserID
		if err != nil {
			dblogger.Error("Failed to read chat rows", zap.Error(err))
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}

		chats = append(chats, c)
		userIDSet[lmUserID] = struct{}{}
		if !c.IsGroup && privateUserID != nil {
			userIDSet[*privateUserID] = struct{}{}
		}
	}

	allUserIDs := make([]int32, 0, len(userIDSet))
	for id := range userIDSet {
		allUserIDs = append(allUserIDs, id)
	}
	if err := rows.Err(); err != nil {
		dblogger.Error("Failed to read chat rows", zap.Error(err))
		return nil, nil, fmt.Errorf("rows error: %w", err)
	}

	dblogger.Info("Chats returns")
	return chats, allUserIDs, nil
}

func (store *DBChatStore) GetOtherChatMembersIdByAuthorId(ctx context.Context, userID int32, chatID int32) ([]domain.MemberWithLastReadMessage, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetOtherChatMembersIdByAuthorId")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `
SELECT 
    cm.member_id,
    cm.last_read_message_id,
    COUNT(m.id) AS unread_count
FROM chat_members cm
LEFT JOIN messages m ON m.chat_id = cm.chat_id 
    AND m.id > cm.last_read_message_id
WHERE cm.member_id != $1 
    AND cm.chat_id = $2
GROUP BY cm.member_id, cm.last_read_message_id
	`
	rows, err := store.db.Query(query, userID, chatID)
	dblogger = dblogger.With(zap.String("query", query), zap.Int32("userID", userID), zap.Int32("chatID", chatID))
	if err != nil {
		dblogger.Error("Failed to find members of chat")
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	members := []domain.MemberWithLastReadMessage{}
	for rows.Next() {
		var member domain.MemberWithLastReadMessage
		err := rows.Scan(
			&member.MemberID,
			&member.LastReadMessageID,
			&member.UnreadCounts,
		)
		if err != nil {
			dblogger.Error("Failed to read member rows", zap.Error(err))
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		dblogger.Error("Failed to read member rows", zap.Error(err))
		return nil, fmt.Errorf("rows error: %w", err)
	}

	dblogger.Info("Members returns")
	return members, nil
}

func (store *DBChatStore) GetFullChatByIDAndSenderID(ctx context.Context, userID int32, chatID int32) (*domain.FullChat, []int32, error) {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetUserFullChats")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	query := `
SELECT
    c.id AS chat_id,
    c.is_group,
    c.name AS chat_name,
    c.avatar AS chat_avatar,

    m.id AS last_message_id,
    m.text AS last_message_text,
    m.created_at AS last_message_created_at,
    m.author_id AS last_message_author_id,

    CASE 
        WHEN NOT c.is_group THEN cm2.member_id
        ELSE null
    END AS private_user_id

FROM chats c
LEFT JOIN messages m ON m.id = (
    SELECT id 
    FROM messages 
    WHERE chat_id = c.id 
    ORDER BY created_at DESC 
    LIMIT 1
)
LEFT JOIN chat_members cm2 
    ON cm2.chat_id = c.id AND cm2.member_id = $2
WHERE c.id = $1;

    `

	dblogger = dblogger.With(zap.String("query", query))

	row := store.db.QueryRow(query, chatID, userID)

	var c domain.FullChat
	var m domain.Message
	var lmUserID int32
	var privateUserID *int32

	err := row.Scan(
		&c.ID,
		&c.IsGroup,
		&c.Name,
		&c.AvatarPath,
		&m.ID,
		&m.Text,
		&m.CreatedAt,
		&lmUserID,
		&privateUserID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("chat not found: %w", err)
		}
		dblogger.Error("Failed to scan chat row", zap.Error(err))
		return nil, nil, fmt.Errorf("scan failed: %w", err)
	}

	m.AuthorID = lmUserID
	m.ChatID = c.ID
	c.LastMessage = m
	c.UserIDWith = privateUserID

	var allUserIDs []int32
	if !c.IsGroup {
		allUserIDs = append(allUserIDs, *privateUserID)
	}
	allUserIDs = append(allUserIDs, lmUserID)
	dblogger.Info("Chat returned successfully", zap.Int32("chat_id", chatID))
	return &c, allUserIDs, nil
}

func (store *DBChatStore) UpdateLastReadMessageByUserIDAndChatID(ctx context.Context, userID, chatID, lastReadMessageID int32) error {
	start := time.Now()
	dblogger := domain.DBLogger(ctx, "chatStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start UpdateLastReadMessageByUserIDAndChatID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	query := `
        UPDATE chat_members
        SET last_read_message_id = $1
        WHERE chat_id = $2
          AND member_id = $3
          AND $1 > last_read_message_id
	`
	_, err := store.db.ExecContext(ctx, query, lastReadMessageID, chatID, userID)
	dblogger = dblogger.With(zap.String("query", query), zap.Int32("userID", userID), zap.Int32("chatID", chatID))
	if err != nil {
		dblogger.Error("Failed update last read message")
		return fmt.Errorf("query failed: %w", err)
	}

	dblogger.Info("LastReadMessage changed")
	return nil
}
