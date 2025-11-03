package db

import (
	"context"
	"database/sql"
	"errors"
	"project/domain"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetMessagesByChatId(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbConn.Close()

	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "created_at"}).
		AddRow(1, 2, 3, "Hello", time.Now()).
		AddRow(2, 3, 3, "World", time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 10, 0).
		WillReturnRows(rows)

	messages, err := store.GetMessagesByChatId(ctx, 3, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, 1, messages[0].ID)
	assert.Equal(t, 2, messages[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbConn.Close()

	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs(2, 3, "Hello").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	messageID, err := store.CreateMessage(ctx, domain.Message{
		AuthorID: 2,
		ChatID:   3,
		Text:     "Hello",
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, messageID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_NoRows(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbConn.Close()

	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "created_at"}))

	messages, err := store.GetMessagesByChatId(ctx, 3, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_Error(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbConn.Close()

	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs(2, 3, "Hello").
		WillReturnError(sql.ErrConnDone)

	messageID, err := store.CreateMessage(ctx, domain.Message{
		AuthorID: 2,
		ChatID:   3,
		Text:     "Hello",
	})
	assert.Error(t, err)
	assert.Equal(t, 0, messageID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_SuccessMultipleRows(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "created_at"}).
		AddRow(1, 1, 3, "Msg1", time.Now()).
		AddRow(2, 2, 3, "Msg2", time.Now()).
		AddRow(3, 1, 3, "Msg3", time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 10, 0).
		WillReturnRows(rows)

	messages, err := store.GetMessagesByChatId(ctx, 3, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestGetMessagesByChatId_RowsScanError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "created_at"}).
		AddRow("notint", 1, 3, "Msg", time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 10, 0).
		WillReturnRows(rows)

	_, err := store.GetMessagesByChatId(ctx, 3, 10, 0)
	assert.Error(t, err)
}

func TestGetMessagesByChatId_QueryError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 10, 0).
		WillReturnError(errors.New("query failed"))

	_, err := store.GetMessagesByChatId(ctx, 3, 10, 0)
	assert.Error(t, err)
}

func TestCreateMessage_Success(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs(1, 2, "Hello").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	id, err := store.CreateMessage(ctx, domain.Message{AuthorID: 1, ChatID: 2, Text: "Hello"})
	assert.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestCreateMessage_QueryError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs(1, 2, "Hello").
		WillReturnError(sql.ErrConnDone)

	id, err := store.CreateMessage(ctx, domain.Message{AuthorID: 1, ChatID: 2, Text: "Hello"})
	assert.Error(t, err)
	assert.Equal(t, 0, id)
}

func TestCreateMessage_ScanError(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text) VALUES ($1, $2, $3) RETURNING id`)).
		WithArgs(1, 2, "Hello").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("notint"))

	id, err := store.CreateMessage(ctx, domain.Message{AuthorID: 1, ChatID: 2, Text: "Hello"})
	assert.Error(t, err)
	assert.Equal(t, 0, id)
}

func TestGetMessagesByChatId_LimitOffsetEdge(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	store := NewDBMessageStore(dbConn)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "created_at"}).
		AddRow(1, 2, 3, "Edge", time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, created_at
              FROM messages
              WHERE chat_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`)).
		WithArgs(3, 1, 0).
		WillReturnRows(rows)

	messages, err := store.GetMessagesByChatId(ctx, 3, 1, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
}
