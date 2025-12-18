package db

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"project/domain"
)

func newMessageStoreMock(t *testing.T) (*DBMessageStore, sqlmock.Sqlmock, *sql.DB) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err, "failed to create sqlmock")
	store := NewDBMessageStore(dbConn).(*DBMessageStore)
	return store, mock, dbConn
}

func TestGetMessagesByChatId_Success(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

	// Mock основного запроса сообщений
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, author_id, chat_id, text, sticker_id, created_at
FROM messages
WHERE chat_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`)).
		WithArgs(chatID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "sticker_id", "created_at"}).
			AddRow(int32(1), int32(100), chatID, "Hello", nil, now).
			AddRow(int32(2), int32(101), chatID, "World", nil, now.Add(-time.Minute)).
			AddRow(int32(3), int32(100), chatID, "Sticker message", int32(5), now.Add(-2*time.Minute)))

	// Mock запроса для вложений первого сообщения
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(int32(1)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("path/to/file1.jpg").
			AddRow("path/to/file2.jpg"))

	// Mock запроса для вложений второго сообщения
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(int32(2)).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("path/to/file3.jpg"))

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)

	// Проверяем первое сообщение с вложениями
	assert.Equal(t, int32(1), messages[0].ID)
	assert.Equal(t, int32(100), messages[0].AuthorID)
	assert.Equal(t, chatID, messages[0].ChatID)
	assert.Equal(t, "Hello", messages[0].Text)
	assert.Nil(t, messages[0].StickerID)
	assert.Equal(t, []string{"path/to/file1.jpg", "path/to/file2.jpg"}, messages[0].Attachments)

	// Проверяем второе сообщение с вложениями
	assert.Equal(t, int32(2), messages[1].ID)
	assert.Equal(t, []string{"path/to/file3.jpg"}, messages[1].Attachments)

	// Проверяем третье сообщение со стикером (без вложений)
	assert.Equal(t, int32(3), messages[2].ID)
	assert.Equal(t, "Sticker message", messages[2].Text)
	assert.Equal(t, int32(5), *messages[2].StickerID)
	assert.Empty(t, messages[2].Attachments)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_NoMessages(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, sticker_id, created_at`)).
		WithArgs(chatID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "sticker_id", "created_at"}))

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	assert.NoError(t, err)
	assert.Empty(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_WithError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, sticker_id, created_at`)).
		WithArgs(chatID, limit, offset).
		WillReturnError(errors.New("database error"))

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_ScanError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)

	// Неправильное количество колонок
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, sticker_id, created_at`)).
		WithArgs(chatID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id"}).
			AddRow(int32(1), int32(100)))

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_RowsError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)

	rows := sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "sticker_id", "created_at"}).
		AddRow(int32(1), int32(100), chatID, "Hello", nil, time.Now()).
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, sticker_id, created_at`)).
		WithArgs(chatID, limit, offset).
		WillReturnRows(rows)

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	assert.Error(t, err)
	assert.Nil(t, messages)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessagesByChatId_AttachmentError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	chatID := int32(1)
	limit := int32(10)
	offset := int32(0)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, author_id, chat_id, text, sticker_id, created_at`)).
		WithArgs(chatID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "chat_id", "text", "sticker_id", "created_at"}).
			AddRow(int32(1), int32(100), chatID, "Hello", nil, now))

	// Mock ошибки при запросе вложений
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(int32(1)).
		WillReturnError(errors.New("attachment error"))

	messages, err := store.GetMessagesByChatId(ctx, chatID, limit, offset)
	// Код продолжает выполнение при ошибке вложений
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Empty(t, messages[0].Attachments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_WithoutAttachments(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{
		AuthorID: int32(100),
		ChatID:   int32(1),
		Text:     "Hello World",
	}
	messageID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text, sticker_id) VALUES ($1, $2, $3, $4) RETURNING id`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, (*int32)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(messageID))
	mock.ExpectCommit()

	id, err := store.CreateMessage(ctx, message)
	assert.NoError(t, err)
	assert.Equal(t, messageID, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_WithAttachments(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{
		AuthorID:    int32(100),
		ChatID:      int32(1),
		Text:        "Hello with attachments",
		Attachments: []string{"path/to/file1.jpg", "path/to/file2.pdf"},
	}
	messageID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text, sticker_id) VALUES ($1, $2, $3, $4) RETURNING id`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, (*int32)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(messageID))

	// Mock для сохранения вложений
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments (obj_id, obj_type, file_path) VALUES ($1, 'message', $2)`)).
		WithArgs(messageID, "path/to/file1.jpg").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments (obj_id, obj_type, file_path) VALUES ($1, 'message', $2)`)).
		WithArgs(messageID, "path/to/file2.pdf").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	id, err := store.CreateMessage(ctx, message)
	assert.NoError(t, err)
	assert.Equal(t, messageID, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_WithSticker(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	stickerID := int32(5)
	message := domain.Message{
		AuthorID:  int32(100),
		ChatID:    int32(1),
		Text:      "Sticker message",
		StickerID: &stickerID,
		// Attachments игнорируются при наличии стикера
		Attachments: []string{"path/to/file.jpg"},
	}
	messageID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages (author_id, chat_id, text, sticker_id) VALUES ($1, $2, $3, $4) RETURNING id`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, message.StickerID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(messageID))
	mock.ExpectCommit()

	id, err := store.CreateMessage(ctx, message)
	assert.NoError(t, err)
	assert.Equal(t, messageID, id)
	// Вложения не должны сохраняться, если есть стикер
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_BeginTxError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{AuthorID: int32(100), ChatID: int32(1), Text: "Hello"}

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	id, err := store.CreateMessage(ctx, message)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_InsertMessageError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{AuthorID: int32(100), ChatID: int32(1), Text: "Hello"}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, (*int32)(nil)).
		WillReturnError(errors.New("insert error"))
	// Rollback будет вызван в defer

	id, err := store.CreateMessage(ctx, message)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.Contains(t, err.Error(), "failed to create message")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_SaveAttachmentsError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{
		AuthorID:    int32(100),
		ChatID:      int32(1),
		Text:        "Hello",
		Attachments: []string{"path/to/file.jpg"},
	}
	messageID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, (*int32)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(messageID))

	// Ошибка при сохранении вложений
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments`)).
		WithArgs(messageID, "path/to/file.jpg").
		WillReturnError(errors.New("attachment error"))

	id, err := store.CreateMessage(ctx, message)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.Contains(t, err.Error(), "failed to save message attachments")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMessage_CommitError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	message := domain.Message{AuthorID: int32(100), ChatID: int32(1), Text: "Hello"}
	messageID := int32(42)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO messages`)).
		WithArgs(message.AuthorID, message.ChatID, message.Text, (*int32)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(messageID))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	id, err := store.CreateMessage(ctx, message)
	assert.Error(t, err)
	assert.Equal(t, int32(0), id)
	assert.Contains(t, err.Error(), "failed to commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageAttachments_Success(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(messageID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).
			AddRow("path/to/file1.jpg").
			AddRow("path/to/file2.pdf").
			AddRow("path/to/file3.png"))

	attachments, err := store.GetMessageAttachments(ctx, messageID)
	assert.NoError(t, err)
	assert.Equal(t, []string{"path/to/file1.jpg", "path/to/file2.pdf", "path/to/file3.png"}, attachments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageAttachments_NoAttachments(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(messageID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}))

	attachments, err := store.GetMessageAttachments(ctx, messageID)
	assert.NoError(t, err)
	assert.Empty(t, attachments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageAttachments_QueryError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(messageID).
		WillReturnError(errors.New("query error"))

	attachments, err := store.GetMessageAttachments(ctx, messageID)
	assert.Error(t, err)
	assert.Nil(t, attachments)
	assert.Contains(t, err.Error(), "failed to query message attachments")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageAttachments_RowsError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	rows := sqlmock.NewRows([]string{"file_path"}).
		AddRow("path/to/file1.jpg").
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(messageID).
		WillReturnRows(rows)

	attachments, err := store.GetMessageAttachments(ctx, messageID)
	assert.Error(t, err)
	assert.Nil(t, attachments)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMessageAttachments_ScanError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	// Неправильное количество колонок
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT file_path
		FROM attachments
		WHERE obj_id = $1 AND obj_type = 'message'
		ORDER BY created_at
	`)).
		WithArgs(messageID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path", "extra_column"}).
			AddRow("path/to/file1.jpg", "extra"))

	attachments, err := store.GetMessageAttachments(ctx, messageID)
	assert.Error(t, err)
	assert.Nil(t, attachments)
	assert.Contains(t, err.Error(), "failed to scan attachment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMessageAttachments_Success(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)
	attachments := []string{"path/to/file1.jpg", "path/to/file2.pdf"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments (obj_id, obj_type, file_path) VALUES ($1, 'message', $2)`)).
		WithArgs(messageID, "path/to/file1.jpg").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments (obj_id, obj_type, file_path) VALUES ($1, 'message', $2)`)).
		WithArgs(messageID, "path/to/file2.pdf").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.SaveMessageAttachments(ctx, messageID, attachments)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMessageAttachments_Empty(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)

	err := store.SaveMessageAttachments(ctx, messageID, []string{})
	assert.NoError(t, err)
	// Никаких запросов не должно быть
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMessageAttachments_BeginTxError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)
	attachments := []string{"path/to/file.jpg"}

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := store.SaveMessageAttachments(ctx, messageID, attachments)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMessageAttachments_InsertError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)
	attachments := []string{"path/to/file.jpg"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments`)).
		WithArgs(messageID, "path/to/file.jpg").
		WillReturnError(errors.New("insert error"))
	// Rollback будет вызван автоматически

	err := store.SaveMessageAttachments(ctx, messageID, attachments)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save message attachment")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveMessageAttachments_CommitError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	messageID := int32(42)
	attachments := []string{"path/to/file.jpg"}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO attachments`)).
		WithArgs(messageID, "path/to/file.jpg").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	err := store.SaveMessageAttachments(ctx, messageID, attachments)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStickerPath_Success(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	stickerID := int32(5)
	expectedPath := "stickers/funny.png"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM stickers WHERE id = $1`)).
		WithArgs(stickerID).
		WillReturnRows(sqlmock.NewRows([]string{"file_path"}).AddRow(expectedPath))

	path, err := store.GetStickerPath(ctx, stickerID)
	assert.NoError(t, err)
	assert.Equal(t, expectedPath, path)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStickerPath_NotFound(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	stickerID := int32(5)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM stickers WHERE id = $1`)).
		WithArgs(stickerID).
		WillReturnError(sql.ErrNoRows)

	path, err := store.GetStickerPath(ctx, stickerID)
	assert.Error(t, err)
	assert.Equal(t, "", path)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStickerPath_QueryError(t *testing.T) {
	store, mock, dbConn := newMessageStoreMock(t)
	defer dbConn.Close()

	ctx := testContext()
	stickerID := int32(5)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_path FROM stickers WHERE id = $1`)).
		WithArgs(stickerID).
		WillReturnError(errors.New("database error"))

	path, err := store.GetStickerPath(ctx, stickerID)
	assert.Error(t, err)
	assert.Equal(t, "", path)
	assert.Contains(t, err.Error(), "failed to get sticker path")
	assert.NoError(t, mock.ExpectationsWereMet())
}
