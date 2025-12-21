package repository

import (
	"github.com/evgeniy-krivenko/grpc-notes/internal/repository/converter"
	"github.com/evgeniy-krivenko/grpc-notes/internal/repository/converter/generated"
	notesrepo "github.com/evgeniy-krivenko/grpc-notes/internal/repository/notes/gen"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/database"
)

var conv converter.Converter = &generated.ConverterImpl{}

type Repo struct {
	notesDB notesrepo.Querier
}

func New(db database.Tx) *Repo {
	return &Repo{
		notesDB: notesrepo.New(db),
	}
}
