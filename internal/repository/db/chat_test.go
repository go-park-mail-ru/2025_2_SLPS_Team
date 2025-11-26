package db

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func newChatStoreMock(t *testing.T) (*DBChatStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	store := NewDBChatStore(dbConn).(*DBChatStore)
	return store, mock, dbConn
}

func TestGetOrCreateChatWithUser_ExistingChat(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	user1 := int32(1)
	user2 := int32(2)
	chatID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT chat_id
        FROM chat_members
        WHERE member_id = ANY($1)
        GROUP BY chat_id
        HAVING COUNT(*) = 2 AND bool_and(member_id = ANY($1))
        LIMIT 1
    `)).
		WithArgs(pq.Array([]int32{1, 2})).
		WillReturnRows(sqlmock.NewRows([]string{"chat_id"}).AddRow(chatID))
	mock.ExpectCommit()

	id, err := store.GetOrCreateChatWithUser(ctx, user1, user2)
	assert.NoError(t, err)
	assert.Equal(t, chatID, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateChatWithUser_NewChat(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	user1 := int32(1)
	user2 := int32(2)
	newChatID := int32(99)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT chat_id
        FROM chat_members
        WHERE member_id = ANY($1)
        GROUP BY chat_id
        HAVING COUNT(*) = 2 AND bool_and(member_id = ANY($1))
        LIMIT 1
    `)).
		WithArgs(pq.Array([]int32{1, 2})).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO chats DEFAULT VALUES RETURNING id`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newChatID))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO chat_members (chat_id, member_id) VALUES ($1, $2), ($1, $3)`)).
		WithArgs(newChatID, int32(1), int32(2)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	id, err := store.GetOrCreateChatWithUser(ctx, user1, user2)
	assert.NoError(t, err)
	assert.Equal(t, newChatID, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsMemberOfChat(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 and member_id = $2)`)).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := store.IsMemberOfChat(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOtherChatMembersIdByAuthorId(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(userID, chatID).
		WillReturnRows(sqlmock.NewRows([]string{"member_id", "last_read_message_id", "unread_count"}).
			AddRow(int32(2), int32(10), int32(3)).
			AddRow(int32(3), int32(15), int32(0)))

	members, err := store.GetOtherChatMembersIdByAuthorId(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, int32(2), members[0].MemberID)
	assert.Equal(t, int32(3), members[1].MemberID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFullChats(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	rows := sqlmock.NewRows([]string{
		"chat_id", "is_group", "chat_name", "chat_avatar",
		"last_message_id", "last_message_text", "last_message_created_at",
		"last_message_author_id", "private_user_id", "unread_count", "last_read_message_id",
	}).AddRow(int32(42), false, "ChatName", "AvatarPath", int32(101), "Hello", time.Now(), int32(2), int32(3), int32(5), int32(100))

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    WHERE cm2.member_id != $1
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
        ELSE 0
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
    `)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	chats, _, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, chats, 1)
	assert.Equal(t, int32(42), chats[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFullChatByIDAndSenderID(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
        ELSE 0
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
    ON cm2.chat_id = c.id AND cm2.member_id != $2
WHERE c.id = $1;
    `)).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"chat_id", "is_group", "chat_name", "chat_avatar",
			"last_message_id", "last_message_text", "last_message_created_at",
			"last_message_author_id", "private_user_id",
		}).AddRow(
			int32(42), false, "ChatName", "AvatarPath",
			int32(101), "Hello", time.Now(), int32(2), int32(3),
		))

	chat, _, err := store.GetFullChatByIDAndSenderID(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.NotNil(t, chat)
	assert.Equal(t, int32(42), chat.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateLastReadMessageByUserIDAndChatID(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)
	lastReadMessageID := int32(100)

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE chat_members
        SET last_read_message_id = $1
        WHERE chat_id = $2
          AND member_id = $3
          AND $1 > last_read_message_id
	`)).
		WithArgs(lastReadMessageID, chatID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}