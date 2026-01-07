package notes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/evgeniy-krivenko/grpc-notes/internal/api/notes/converter"
	"github.com/evgeniy-krivenko/grpc-notes/internal/api/notes/converter/generated"
	"github.com/evgeniy-krivenko/grpc-notes/internal/ctxtr"
	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	v1 "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
)

var _ grpcx.Service = (*Service)(nil)

var conv converter.Converter = &generated.ConverterImpl{}

type notesUsecase interface {
	CreateNote(ctx context.Context, userID int64, title, content string) (entity.Note, error)
	GetNote(ctx context.Context, id int64) (entity.Note, error)
	GetNotesByUserID(ctx context.Context, userID int64) ([]entity.Note, error)
	DeleteNote(ctx context.Context, id int64) error
	SubscribeToEvents(ctx context.Context, userID int64) (<-chan entity.CreateNoteEvent, error)
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
	if req.GetNoteId() == 1 {
		return nil, withNoteError(
			codes.FailedPrecondition,
			v1.ErrorCode_ERROR_CODE_INVALID_TEXT,
		)
	}

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

func withNoteError(code codes.Code, reason v1.ErrorCode) error {
	st := status.New(code, "get note error")

	noteErr := &v1.NoteError{Reason: reason}

	st, err := st.WithDetails(noteErr)
	if err != nil {
		return err
	}

	return st.Err()
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

func (s *Service) SubscribeToEvents(req *v1.SubscribeToEventRequest, stream v1.NoteAPI_SubscribeToEventsServer) error {
	ctx := stream.Context()

	slogx.Info(ctx, "client subscribe to events", slogx.UserId(req.UserId))

	if err := sendHealthCheck(ctx, stream); err != nil {
		return fmt.Errorf("send first health check: %v", err)
	}

	events, err := s.usecase.SubscribeToEvents(ctx, req.GetUserId())
	if err != nil {
		return fmt.Errorf("get events: %v", err)
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slogx.Debug(ctx, "context canceled")
			return nil

		case <-ticker.C:
			if err := sendHealthCheck(ctx, stream); err != nil {
				return err
			}
		case event := <-events:

			n := conv.ConvertNoteToProto(event.CreatedNote)
			note := v1.SubscribeToEventResponse_CreatedNote{CreatedNote: n}
			resp := v1.SubscribeToEventResponse{Result: &note}

			if err := stream.Send(&resp); err != nil {
				code := status.Code(err)

				switch code {
				case codes.Canceled | codes.DeadlineExceeded:
					slogx.Info(ctx, "client unsubscribe", slogx.GrpcCode(code))
					return nil
				case codes.Unavailable:
					slogx.Warn(ctx, "client unavailable")
					return nil
				default:
					slogx.Error(ctx, "unexpected send error", slogx.Err(err), slogx.GrpcCode(code))
					return err
				}
			}
		}
	}
}

func (s *Service) UploadMetrics(stream v1.NoteAPI_UploadMetricsServer) error {
	ctx := stream.Context()

	var sum v1.SummaryResponse
	for {
		if ctx.Err() != nil {
			slogx.Info(ctx, "context canceled")
			break
		}

		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				slogx.Info(ctx, "client close stream")
				break
			}

			return err
		}

		sum.TotalView += resp.NoteViewCounter
		slogx.Info(ctx, "receive metrics from client")
	}

	if err := stream.SendAndClose(&sum); err != nil {
		return err
	}

	return nil
}

func (s *Service) Chat(stream v1.NoteAPI_ChatServer) error {
	ctx := stream.Context()

	messages := getMessages()
	eg, ctx := errgroup.WithContext(ctx)

	ackChan := make(chan string)

	eg.Go(func() error {
		defer close(ackChan)

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					slogx.Info(ctx, "client closed stream")
					return nil
				}

				return fmt.Errorf("receive msg: %v", err)
			}

			correlationID := msg.CorrelationId

			eg.Go(func() error {
				select {
				case <-ctx.Done():
					return nil
				case ackChan <- correlationID:
				}

				return nil
			})

			slogx.Info(ctx, "receive msg from client", slog.String("msg", msg.GetContent()))
		}
	})

	eg.Go(func() error {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slogx.Info(ctx, "context canceled in send gorutine")
				return nil
			case <-ticker.C:
			}

			idx := rand.Intn(10)
			msg := messages[idx]

			if err := stream.Send(&v1.ServerMessage{
				Content: msg,
			}); err != nil {
				return fmt.Errorf("send msg: %v", err)
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				slogx.Info(ctx, "context canceled in ack gorutine")
				return nil
			case correlationID, ok := <-ackChan:
				if !ok {
					return nil
				}

				msg := v1.ServerMessage{IsAck: true, CorrelationId: correlationID}

				if err := stream.Send(&msg); err != nil {
					slogx.Error(ctx, "while ack client message", slog.String("correlation_id", correlationID))

					continue
				}

				slogx.Info(ctx, "success to ack message", slog.String("correlation_id", correlationID))
			}
		}
	})

	if err := eg.Wait(); err != nil && err != context.Canceled {
		return fmt.Errorf("stream waiting: %v", err)
	}

	return nil
}

func getMessages() map[int]string {
	phrases := []string{
		"hello",
		"how are you?",
		"how does going?",
		"okey",
		"stay in touch",
		"nice to meet you",
		"good morning",
		"afternoon!",
		"hi, fellas!",
		"hello, people!",
	}

	messages := make(map[int]string, 10)

	for idx, phrase := range phrases {
		messages[idx] = phrase
	}

	return messages
}

func sendHealthCheck(_ context.Context, stream v1.NoteAPI_SubscribeToEventsServer) error {
	healthCheck := &v1.HealthCheck{Timestamp: converter.ConvertTimeToDateTime(time.Now())}
	hc := &v1.SubscribeToEventResponse_HealthCheck{HealthCheck: healthCheck}

	if err := stream.Send(&v1.SubscribeToEventResponse{Result: hc}); err != nil {
		return fmt.Errorf("send health check message: %v", err)
	}

	return nil
}
