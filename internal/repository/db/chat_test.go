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
	user1 := 1
	user2 := 2
	chatID := 42

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT chat_id
        FROM chat_members
        WHERE member_id = ANY($1)
        GROUP BY chat_id
        HAVING COUNT(*) = 2 AND bool_and(member_id = ANY($1))
        LIMIT 1
    `)).
		WithArgs(pq.Array([]int{1, 2})).
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
	user1 := 1
	user2 := 2
	newChatID := 99

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT chat_id
        FROM chat_members
        WHERE member_id = ANY($1)
        GROUP BY chat_id
        HAVING COUNT(*) = 2 AND bool_and(member_id = ANY($1))
        LIMIT 1
    `)).
		WithArgs(pq.Array([]int{1, 2})).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO chats DEFAULT VALUES RETURNING id`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newChatID))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO chat_members (chat_id, member_id) VALUES ($1, $2), ($1, $3)`)).
		WithArgs(newChatID, 1, 2).
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
	userID := 1
	chatID := 42

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
	userID := 1
	chatID := 42

	mock.ExpectQuery(regexp.QuoteMeta(`
	SELECT member_id
	FROM chat_members
	WHERE member_id != $1 and chat_id = $2
	`)).
		WithArgs(userID, chatID).
		WillReturnRows(sqlmock.NewRows([]string{"member_id"}).AddRow(2).AddRow(3))

	members, err := store.GetOtherChatMembersIdByAuthorId(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.Equal(t, []int{2, 3}, members)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFullChats(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := 1
	limit := 10
	offset := 0

	rows := sqlmock.NewRows([]string{
		"chat_id", "is_group", "chat_name", "chat_avatar",
		"last_message_id", "last_message_text", "last_message_created_at",
		"last_message_author_id", "last_message_author_name", "last_message_author_avatar",
	}).AddRow(42, false, "ChatName", "AvatarPath", 101, "Hello", time.Now(), 2, "User Two", "UserAvatar")

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	chats, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, chats, 1)
	assert.Equal(t, 42, chats[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}
