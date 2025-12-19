package db

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newChatStoreMock(t *testing.T) (*DBChatStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
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

func TestGetOrCreateChatWithUser_WithErrorOnBegin(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	id, err := store.GetOrCreateChatWithUser(ctx, 1, 2)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateChatWithUser_WithErrorOnChatCheck(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT chat_id`)).
		WithArgs(pq.Array([]int32{1, 2})).
		WillReturnError(errors.New("query error"))
	mock.ExpectRollback()

	id, err := store.GetOrCreateChatWithUser(ctx, 1, 2)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateChatWithUser_WithErrorOnCreateChat(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT chat_id`)).
		WithArgs(pq.Array([]int32{1, 2})).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO chats DEFAULT VALUES RETURNING id`)).
		WillReturnError(errors.New("insert error"))
	mock.ExpectRollback()

	id, err := store.GetOrCreateChatWithUser(ctx, 1, 2)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateChatWithUser_WithErrorOnInsertMembers(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	newChatID := int32(99)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT chat_id`)).
		WithArgs(pq.Array([]int32{1, 2})).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO chats DEFAULT VALUES RETURNING id`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newChatID))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO chat_members`)).
		WithArgs(newChatID, int32(1), int32(2)).
		WillReturnError(errors.New("insert members error"))
	mock.ExpectRollback()

	id, err := store.GetOrCreateChatWithUser(ctx, 1, 2)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsMemberOfChat_True(t *testing.T) {
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

func TestIsMemberOfChat_False(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 and member_id = $2)`)).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	ok, err := store.IsMemberOfChat(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsMemberOfChat_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id = $1 and member_id = $2)`)).
		WithArgs(chatID, userID).
		WillReturnError(errors.New("query error"))

	ok, err := store.IsMemberOfChat(ctx, userID, chatID)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsChatExist_True(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1 )`)).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.IsChatExist(ctx, chatID)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsChatExist_False(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1 )`)).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := store.IsChatExist(ctx, chatID)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsChatExist_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1 )`)).
		WithArgs(chatID).
		WillReturnError(errors.New("query error"))

	exists, err := store.IsChatExist(ctx, chatID)
	assert.Error(t, err)
	assert.False(t, exists)
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
	assert.Equal(t, int32(10), members[0].LastReadMessageID)
	assert.Equal(t, int32(3), members[0].UnreadCounts)
	assert.Equal(t, int32(3), members[1].MemberID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOtherChatMembersIdByAuthorId_EmptyResult(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, chatID).
		WillReturnRows(sqlmock.NewRows([]string{"member_id", "last_read_message_id", "unread_count"}))

	members, err := store.GetOtherChatMembersIdByAuthorId(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.Empty(t, members)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOtherChatMembersIdByAuthorId_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(userID, chatID).
		WillReturnError(errors.New("query error"))

	members, err := store.GetOtherChatMembersIdByAuthorId(ctx, userID, chatID)
	assert.Error(t, err)
	assert.Nil(t, members)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFullChats(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"chat_id", "is_group", "chat_name", "chat_avatar",
		"last_message_id", "last_message_text", "last_message_created_at",
		"last_message_author_id", "private_user_id", "unread_count", "last_read_message_id",
	}).
		AddRow(int32(42), false, "ChatName", "AvatarPath", int32(101), "Hello", now, int32(2), int32(3), int32(5), int32(100)).
		AddRow(int32(43), true, "GroupChat", "GroupAvatar", int32(102), "Group msg", now, int32(4), int32(0), int32(0), int32(99))

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
    `)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	chats, userIDs, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, chats, 2)
	assert.Len(t, userIDs, 3) // 2 + 3 + 4 = 3 unique IDs

	// Check first chat (private)
	assert.Equal(t, int32(42), chats[0].ID)
	assert.False(t, chats[0].IsGroup)
	assert.Equal(t, int32(101), chats[0].LastMessage.ID)
	assert.Equal(t, int32(2), chats[0].LastMessage.AuthorID)
	assert.Equal(t, int32(3), *chats[0].UserIDWith)
	assert.Equal(t, int32(5), chats[0].UnreadCounts)
	assert.Equal(t, int32(100), chats[0].LastReadMessageID)

	// Check second chat (group)
	assert.Equal(t, int32(43), chats[1].ID)
	assert.True(t, chats[1].IsGroup)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFullChats_EmptyResult(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`WITH last_messages AS`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{
			"chat_id", "is_group", "chat_name", "chat_avatar",
			"last_message_id", "last_message_text", "last_message_created_at",
			"last_message_author_id", "private_user_id", "unread_count", "last_read_message_id",
		}))

	chats, userIDs, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, chats)
	assert.Empty(t, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserFullChats_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`WITH last_messages AS`)).
		WithArgs(userID, limit, offset).
		WillReturnError(errors.New("query error"))

	chats, userIDs, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, chats)
	assert.Nil(t, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFullChatByIDAndSenderID(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)
	now := time.Now()

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
    `)).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"chat_id", "is_group", "chat_name", "chat_avatar",
			"last_message_id", "last_message_text", "last_message_created_at",
			"last_message_author_id", "private_user_id",
		}).AddRow(
			int32(42), false, "ChatName", "AvatarPath",
			int32(101), "Hello", now, int32(2), int32(3),
		))

	chat, userIDs, err := store.GetFullChatByIDAndSenderID(ctx, userID, chatID)
	assert.NoError(t, err)
	assert.NotNil(t, chat)
	assert.Equal(t, int32(42), chat.ID)
	assert.False(t, chat.IsGroup)
	assert.Equal(t, int32(101), chat.LastMessage.ID)
	assert.Equal(t, int32(2), chat.LastMessage.AuthorID)
	assert.Equal(t, int32(3), *chat.UserIDWith)
	assert.Len(t, userIDs, 2) // 2 (author) + 3 (private user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFullChatByIDAndSenderID_NotFound(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(chatID, userID).
		WillReturnError(sql.ErrNoRows)

	chat, userIDs, err := store.GetFullChatByIDAndSenderID(ctx, userID, chatID)
	assert.Error(t, err)
	assert.Nil(t, chat)
	assert.Nil(t, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFullChatByIDAndSenderID_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs(chatID, userID).
		WillReturnError(errors.New("query error"))

	chat, userIDs, err := store.GetFullChatByIDAndSenderID(ctx, userID, chatID)
	assert.Error(t, err)
	assert.Nil(t, chat)
	assert.Nil(t, userIDs)
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

func TestUpdateLastReadMessageByUserIDAndChatID_NoRowsAffected(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)
	lastReadMessageID := int32(100)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE chat_members`)).
		WithArgs(lastReadMessageID, chatID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID)
	assert.NoError(t, err) // No error even if no rows affected
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateLastReadMessageByUserIDAndChatID_WithError(t *testing.T) {
	store, mock, dbConn := newChatStoreMock(t)
	defer dbConn.Close()

	ctx := context.Background()
	userID := int32(1)
	chatID := int32(42)
	lastReadMessageID := int32(100)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE chat_members`)).
		WithArgs(lastReadMessageID, chatID, userID).
		WillReturnError(errors.New("exec error"))

	err := store.UpdateLastReadMessageByUserIDAndChatID(ctx, userID, chatID, lastReadMessageID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Тест на проверку сортировки ID в GetOrCreateChatWithUser
func TestGetOrCreateChatWithUser_IDOrdering(t *testing.T) {
	tests := []struct {
		name     string
		user1    int32
		user2    int32
		expected []int32
	}{
		{"Already sorted", 1, 2, []int32{1, 2}},
		{"Reverse order", 2, 1, []int32{1, 2}},
		{"Equal IDs", 5, 5, []int32{5, 5}},
		{"Large numbers", 100, 50, []int32{50, 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock, dbConn := newChatStoreMock(t)
			defer dbConn.Close()

			ctx := context.Background()
			chatID := int32(42)

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT chat_id`)).
				WithArgs(pq.Array(tt.expected)).
				WillReturnRows(sqlmock.NewRows([]string{"chat_id"}).AddRow(chatID))
			mock.ExpectCommit()

			id, err := store.GetOrCreateChatWithUser(ctx, tt.user1, tt.user2)
			assert.NoError(t, err)
			assert.Equal(t, chatID, id)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Тест на ошибку при закрытии rows в GetUserFullChats
func TestGetUserFullChats_RowsError(t *testing.T) {
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
	}).AddRow(int32(42), false, "ChatName", "AvatarPath", int32(101), "Hello", time.Now(), int32(2), int32(3), int32(5), int32(100)).
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`WITH last_messages AS`)).
		WithArgs(userID, limit, offset).
		WillReturnRows(rows)

	chats, userIDs, err := store.GetUserFullChats(ctx, userID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, chats)
	assert.Nil(t, userIDs)
	assert.NoError(t, mock.ExpectationsWereMet())
}
