package notes

import (
	v1 "github.com/evgeniy-krivenko/grpc-notes/pkg/api/notes/v1"
	"github.com/evgeniy-krivenko/grpc-notes/pkg/grpcx"
	"google.golang.org/grpc"
)

var _ grpcx.Service = (*Service)(nil)

type Service struct {
	v1.UnimplementedNoteAPIServer
}

// RegisterService implements grpcx.Service.
func (s *Service) RegisterService(grpc.ServiceRegistrar) {
	panic("unimplemented")
}

func New() *Service {
	return nil
}
