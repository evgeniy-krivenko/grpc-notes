package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/logger/slogx"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := slogx.InitGlobal(
		os.Stdout,
		"info",
		true,
	); err != nil {
		log.Fatalf("init logger: %v", err)
	}

	conn, err := grpc.NewClient(
		"127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("new client conn: %v", err)
	}
	defer conn.Close()

	c := pb.NewNoteAPIClient(conn)

	md := metadata.New(map[string]string{"authorization": grpcx.MockToken})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := c.GetNote(ctx, &pb.GetNoteRequest{NoteId: 1})
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
