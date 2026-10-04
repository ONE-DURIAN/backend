package uid

import (
	"github.com/google/uuid"
)

// NewV7 generates a time-ordered UUIDv7 as a string.
// UUIDv7 embeds a millisecond timestamp in the high bits, ensuring that records
// are stored sequentially in PostgreSQL B-Tree indexes without page fragmentation.
func NewV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to random UUID if clock/entropy error occurs
		return uuid.NewString()
	}
	return id.String()
}

// IsValid verifies whether a given string is a valid UUID format.
func IsValid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
