package dbRedis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"project/domain"
	"project/internal/service"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

type RedisSessionStore struct {
	redisConn redis.Conn
}

func NewRedisSessionStore(conn redis.Conn) domain.SessionStore {
	return &RedisSessionStore{
		redisConn: conn,
	}
}

const sessionTTL = 86400

func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)

	cryptoReader := rand.Reader
	_, err := cryptoReader.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

func (store *RedisSessionStore) AddSession(ctx context.Context, userID int) (*domain.SIDAndSCRFToken, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "sessionStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start AddSession")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	ID, err := generateRandomToken()
	if err != nil {
		dblogger.Error("Failed to to generate session ID", zap.Error(err))
		return nil, err
	}
	CSRFToken, err := generateRandomToken()
	if err != nil {
		dblogger.Error("Failed to to generate session ID", zap.Error(err))
		return nil, err
	}
	session := domain.Session{UserID: userID, CSRFToken: CSRFToken}
	dataSerialized, err := json.Marshal(session)
	if err != nil {
		dblogger.Error("Failed to marshal session", zap.Error(err))
		return nil, err
	}
	mkey := "sessions:" + ID
	result, err := redis.String(store.redisConn.Do("SET", mkey, dataSerialized, "EX", sessionTTL))
	if err != nil {
		dblogger.Error("Failed to add session", zap.Error(err))
		return nil, err
	}

	if result != "OK" {
		dblogger.Error("Failed to add session")
		return nil, fmt.Errorf("result not OK")
	}
	tokens := domain.SIDAndSCRFToken{
		SID:       ID,
		CSRFToken: CSRFToken,
	}
	dblogger.Info("Session added")
	return &tokens, nil
}

func (store *RedisSessionStore) GetSessionBySessionID(ctx context.Context, sessionID string) (*domain.Session, error) {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "sessionStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start GetSessionBySessionID")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()

	mkey := "sessions:" + sessionID
	data, err := redis.Bytes(store.redisConn.Do("GET", mkey))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			dblogger.Info("session not found")
			return nil, domain.ErrNotFound
		}
		dblogger.Error("Failed to read session from dbRedis", zap.Error(err))
		return nil, err
	}
	sess := &domain.Session{}
	err = json.Unmarshal(data, sess)
	if err != nil {
		dblogger.Error("Failed to unpack session", zap.Error(err))
		return nil, err
	}
	return sess, nil
}

func (store *RedisSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	start := time.Now()
	dblogger := service.DBLogger(ctx, "sessionStore")
	dbloggerCopy := dblogger
	dbloggerCopy.Info("DB start DeleteSession")

	defer func() {
		duration := time.Since(start)
		dbloggerCopy.Info("DB operation finished", zap.Duration("duration", duration))
	}()
	mkey := "sessions:" + sessionID
	_, err := redis.Int(store.redisConn.Do("DEL", mkey))
	if err != nil {
		dblogger.Error("Failed to delete session from dbRedis", zap.Error(err))
		return err
	}
	return nil
}
