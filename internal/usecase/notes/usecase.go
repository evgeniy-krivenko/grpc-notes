package notes

import (
	"context"
	"fmt"

	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
)

type notesRepository interface {
	CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error)
	GetNote(ctx context.Context, id int64) (entity.Note, error)
	GetNotesByUserID(ctx context.Context, userID int64) ([]entity.Note, error)
	DeleteNote(ctx context.Context, id int64) error
}

//go:generate go run github.com/kazhuravlev/options-gen/cmd/options-gen@v0.55.2 -out-filename=usecase_options.gen.go -from-struct=Options
type Options struct {
	repo notesRepository `option:"mandatory" validate:"required"`
}

type Usecase struct {
	Options
}

func New(opts Options) (*Usecase, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate notes usecase options: %v", err)
	}

	return &Usecase{Options: opts}, nil
}

func (u *Usecase) CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error) {
	note, err := u.repo.CreateNote(ctx, userID, title, content)
	if err != nil {
		return entity.Note{}, fmt.Errorf("usecase create note: %w", err)
	}

    slogx.Info(ctx, "success to create note", slogx.UserId(userID))
	return note, nil
}

func (u *Usecase) GetNote(ctx context.Context, id int64) (entity.Note, error) {
	note, err := u.repo.GetNote(ctx, id)
	if err != nil {
		return entity.Note{}, fmt.Errorf("usecase get note: %w", err)
	}

	return note, nil
}

func (u *Usecase) GetNotesByUserID(ctx context.Context, userID int64) ([]entity.Note, error) {
	notes, err := u.repo.GetNotesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase get notes by user: %w", err)
	}

	return notes, nil
}

func (u *Usecase) DeleteNote(ctx context.Context, id int64) error {
	if err := u.repo.DeleteNote(ctx, id); err != nil {
		return fmt.Errorf("usecase delete note: %w", err)
	}

	return nil
}
