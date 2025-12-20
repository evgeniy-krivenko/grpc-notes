package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	notesrepo "github.com/evgeniy-krivenko/grpc-notes/internal/repository/notes/gen"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
)

func (r *Repo) CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error) {
	row, err := r.notesDB.CreateNote(ctx, notesrepo.CreateNoteParams{
		UserID:  userID,
		Title:   title,
		Content: content,
	})
	if err != nil {
		return entity.Note{}, fmt.Errorf("create note: %v", err)
	}

    slogx.Debug(ctx, "success to create note", slogx.UserId(userID))

	return conv.ConvertNoteToEntity(row), nil
}

func (r *Repo) GetNote(ctx context.Context, id int64) (entity.Note, error) {
	row, err := r.notesDB.GetNote(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Note{}, entity.ErrNoteNotFound
		}
		return entity.Note{}, fmt.Errorf("get note: %v", err)
	}

	return conv.ConvertNoteToEntity(row), nil
}

func (r *Repo) GetNotesByUserID(ctx context.Context, userID int64) ([]entity.Note, error) {
	rows, err := r.notesDB.GetNotesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get notes by user: %v", err)
	}

	return conv.ConvertNotesToEntity(rows), nil
}

func (r *Repo) DeleteNote(ctx context.Context, id int64) error {
	if err := r.notesDB.DeleteNote(ctx, id); err != nil {
		return fmt.Errorf("delete note: %v", err)
	}

	return nil
}
