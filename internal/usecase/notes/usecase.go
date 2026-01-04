package notes

import (
	"context"
	"fmt"

	"github.com/imkira/go-observer"

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
	observer observer.Property
}

func New(opts Options) (*Usecase, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate notes usecase options: %v", err)
	}

	prop := observer.NewProperty(entity.Note{})

	return &Usecase{Options: opts, observer: prop}, nil
}

func (u *Usecase) CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error) {
	note, err := u.repo.CreateNote(ctx, userID, title, content)
	if err != nil {
		return entity.Note{}, fmt.Errorf("usecase create note: %w", err)
	}

	u.observer.Update(note)

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

func (u *Usecase) SubscribeToEvents(ctx context.Context, userID int64) (<-chan entity.CreateNoteEvent, error) {
	// ignore user id for simplicity

	stream := u.observer.Observe()

	result := make(chan entity.CreateNoteEvent)
	go func() {
		defer close(result)
		for {
			select {
			case <-ctx.Done():
				return

			case <-stream.Changes():
				note := stream.Next().(entity.Note)

				select {
				case <-ctx.Done():
					return
				case result <- entity.CreateNoteEvent{CreatedNote: note}:
				}
			}
		}
	}()

	return result, nil
}
