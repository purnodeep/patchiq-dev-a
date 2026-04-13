package workflow

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgUUID parses a string UUID into pgtype.UUID.
func pgUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	parsed, err := uuid.Parse(s)
	if err != nil {
		return u
	}
	u.Bytes = parsed
	u.Valid = true
	return u
}

// uuidStr converts pgtype.UUID to string.
func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
