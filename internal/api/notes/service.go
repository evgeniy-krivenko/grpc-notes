package notes

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/evgeniy-krivenko/grpc-notes/internal/api/notes/converter"
	"github.com/evgeniy-krivenko/grpc-notes/internal/api/notes/converter/generated"
	"github.com/evgeniy-krivenko/grpc-notes/internal/ctxtr"
	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	v1 "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
)

var _ grpcx.Service = (*Service)(nil)

var conv converter.Converter = &generated.ConverterImpl{}

type notesUsecase interface {
	CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error)
	GetNote(ctx context.Context, id int64) (entity.Note, error)
	GetNotesByUserID(ctx context.Context, userID int64) ([]entity.Note, error)
	DeleteNote(ctx context.Context, id int64) error
}

//go:generate go run github.com/kazhuravlev/options-gen/cmd/options-gen@v0.33.2 -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	usecase notesUsecase `option:"mandatory" validate:"required"`
}

type Service struct {
	v1.UnimplementedNoteAPIServer
	Options
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate notes service options: %v", err)
	}

	return &Service{Options: opts}, nil
}

func (s *Service) RegisterService(srv grpc.ServiceRegistrar) {
	v1.RegisterNoteAPIServer(srv, s)
}

func (s *Service) CreateNote(ctx context.Context, req *v1.CreateNoteRequest) (*v1.CreateNoteResponse, error) {
	userID, err := ctxtr.UserID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "create note: %v", err)
	}

	note, err := s.usecase.CreateNote(ctx, userID, req.Title, req.Content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create note: %v", err)
	}

	protoNote := conv.ConvertNoteToProto(note)

	return &v1.CreateNoteResponse{Note: protoNote}, nil
}

func (s *Service) GetNote(ctx context.Context, req *v1.GetNoteRequest) (*v1.GetNoteResponse, error) {
	note, err := s.usecase.GetNote(ctx, req.NoteId)
	if err != nil {
		if errors.Is(err, entity.ErrNoteNotFound) {
			return nil, status.Error(codes.NotFound, "note not found")
		}
		return nil, status.Errorf(codes.Internal, "get note: %v", err)
	}

	return &v1.GetNoteResponse{
		Note: conv.ConvertNoteToProto(note),
	}, nil
}

func (s *Service) GetNotes(ctx context.Context, req *v1.GetNotesRequest) (*v1.GetNotesResponse, error) {
	notes, err := s.usecase.GetNotesByUserID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get notes: %v", err)
	}

	return &v1.GetNotesResponse{
		Notes: conv.ConvertNotesToProto(notes),
	}, nil
}

func (s *Service) DeleteNote(ctx context.Context, req *v1.DeleteNoteRequest) (*v1.DeleteNoteResponse, error) {
	if err := s.usecase.DeleteNote(ctx, req.NoteId); err != nil {
		return nil, status.Errorf(codes.Internal, "delete note: %v", err)
	}

	return &v1.DeleteNoteResponse{}, nil
}
