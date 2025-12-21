package entity

import (
	"errors"
	"time"
)

var ErrNoteNotFound = errors.New("note not found")

type Note struct {
	ID        int64
	UserID    int64
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
