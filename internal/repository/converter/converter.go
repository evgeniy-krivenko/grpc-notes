package converter

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	notesrepo "github.com/evgeniy-krivenko/grpc-notes/internal/repository/notes/gen"
)

// goverter:converter
// goverter:output:file ./generated/generated.go
// goverter:output:package generated
// goverter:extend ConvertTimestampzToTime
// goverter:extend ConvertTimeToTimestampz
// goverter:skipCopySameType
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.7.0 gen .
type Converter interface {
	ConvertNoteToEntity(row notesrepo.Note) entity.Note
	ConvertNotesToEntity(rows []notesrepo.Note) []entity.Note
}

func ConvertTimestampzToTime(t pgtype.Timestamptz) time.Time {
	return t.Time
}

func ConvertTimeToTimestampz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
