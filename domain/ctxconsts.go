package domain

type contextKey string

const UserIDKey = contextKey("userID")
const LoggerKey = contextKey("logger")
const TempSessionCtxKey contextKey = "tempSessionInfo"
