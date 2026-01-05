package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	if err := slogx.InitGlobal(
		os.Stdout,
		"info",
		true,
	); err != nil {
		return fmt.Errorf("init logger: %v", err)
	}

	conn, err := grpc.NewClient(
		"127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("new client conn: %v", err)
	}
	defer conn.Close()

	c := pb.NewNoteAPIClient(conn)

	// getNote(ctx, c)
	// subscribeToEvents(ctx, c)
	// if err := sendMetrics(ctx, c); err != nil {
	// 	return err
	// }

	if err := chatWithServer(ctx, c); err != nil {
		return err
	}

	return nil
}

func subscribeToEvents(ctx context.Context, client pb.NoteAPIClient) error {
	req := pb.SubscribeToEventRequest{UserId: 1}

	streamer, err := client.SubscribeToEvents(ctx, &req)
	if err != nil {
		return fmt.Errorf("subscribe to events: %v", err)
	}

	for {
		if ctx.Err() != nil {
			slogx.Info(ctx, "context canceled")
			return nil
		}

		resp, err := streamer.Recv()
		if err != nil {
			if err == io.EOF {
				slogx.Info(ctx, "server closed stream")
				return nil
			}

			return fmt.Errorf("subscribe to events recv: %v", err)
		}

		switch r := resp.Result.(type) {
		case *pb.SubscribeToEventResponse_HealthCheck:
			log.Printf("server sent health check")
		case *pb.SubscribeToEventResponse_CreatedNote:
			log.Printf("response of created note: %v", r.CreatedNote)
		}
	}
}

func getNote(ctx context.Context, client pb.NoteAPIClient) {
	md := metadata.New(map[string]string{"authorization": grpcx.MockToken})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := client.GetNote(ctx, &pb.GetNoteRequest{NoteId: 1})
	if err != nil {
		if noteErr, ok := NoteError(err); ok {
			slogx.Error(ctx, "note error", slog.String(
				"reason",
				noteErr.Reason.String(),
			))
		} else {
			slogx.Error(ctx, "unknown err", slogx.Err(err))
		}
	}

	slogx.Info(ctx, "get note success", slog.Any("response", resp))
}

func sendMetrics(ctx context.Context, client pb.NoteAPIClient) error {
	stream, err := client.UploadMetrics(ctx)
	if err != nil {
		return fmt.Errorf("send metrics: %v", err)
	}

	for i := range 10 {
		if err := stream.Send(&pb.MetricsRequest{NoteViewCounter: int64(i)}); err != nil {
			return err
		}

		slogx.Info(ctx, "send metrics to server")
	}

	reps, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	slogx.Info(ctx, "receive summary from server", slog.Any("summary", reps))
	return nil
}

func chatWithServer(ctx context.Context, client pb.NoteAPIClient) error {
	stream, err := client.Chat(ctx)
	if err != nil {
		return fmt.Errorf("chat with server: %v", err)
	}

	eg, ctx := errgroup.WithContext(stream.Context())

	messages := getMessages()

	eg.Go(func() error {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		defer stream.CloseSend()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}

			correlationID := uuid.NewString()
			idx := rand.Intn(10)
			msg := messages[idx]

			clientMsg := pb.Message{
				CorrelationId: correlationID,
				Content:       msg,
			}

			if err := stream.Send(&clientMsg); err != nil {
				if err == io.EOF {
					slogx.Info(ctx, "server closed stream")
					return nil
				}

				slogx.Error(ctx, "send msg to server", slogx.Err(err))
			}
		}
	})

	eg.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					slogx.Info(ctx, "server closed stream")
					return nil
				}

				slogx.Error(ctx, "receive msg from server", slogx.Err(err))
			}

			if msg.IsAck {
				slogx.Info(ctx, "ack message", slog.String("correlation_id", msg.GetCorrelationId()))
			} else {
				slogx.Info(ctx, "get message from server", slog.String("msg", msg.Content))
			}
		}
	})

	return eg.Wait()
}

func NoteError(err error) (*pb.NoteError, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}

	for _, s := range st.Details() {
		if noteErr, ok := s.(*pb.NoteError); ok {
			return noteErr, true
		}
	}

	return nil, false
}
