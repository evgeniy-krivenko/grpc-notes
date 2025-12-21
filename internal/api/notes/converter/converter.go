package converter

import (
	"time"

	"google.golang.org/genproto/googleapis/type/datetime"

	"github.com/evgeniy-krivenko/grpc-notes/internal/entity"
	v1 "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
)

// goverter:converter
// goverter:output:file ./generated/generated.go
// goverter:output:package generated
// goverter:extend ConvertTimeToDateTime
// goverter:extend ConvertDateTimeToTime
// goverter:skipCopySameType
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.7.0 gen .
type Converter interface {
	// goverter:map ID Id
	// goverter:map UserID UserId
	// goverter:map CreatedAt CreatedAt | ConvertTimeToDateTime
	// goverter:map UpdatedAt UpdatedAt | ConvertTimeToDateTime
	ConvertNoteToProto(note entity.Note) *v1.Note

	ConvertNotesToProto(notes []entity.Note) []*v1.Note
}

func ConvertTimeToDateTime(t time.Time) *datetime.DateTime {
	if t.IsZero() {
		return nil
	}

	return &datetime.DateTime{
		Year:    int32(t.Year()),
		Month:   int32(t.Month()),
		Day:     int32(t.Day()),
		Hours:   int32(t.Hour()),
		Minutes: int32(t.Minute()),
		Seconds: int32(t.Second()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func ConvertDateTimeToTime(dt *datetime.DateTime) time.Time {
	if dt == nil {
		return time.Time{}
	}

	return time.Date(
		int(dt.Year),
		time.Month(dt.Month),
		int(dt.Day),
		int(dt.Hours),
		int(dt.Minutes),
		int(dt.Seconds),
		int(dt.Nanos),
		time.UTC,
	)
}
