package server

import (
	context "context"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Exchange(ctx context.Context, in *ExchangeRequest) (*ExchangeReply, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "exchange was cancelled")

	default:
		return s.exchangeInternal(in)
	}
}

func (s *Server) exchangeInternal(in *ExchangeRequest) (*ExchangeReply, error) {
	s.consumerMu.RLock()
	if _, exists := s.consumers[in.PublicKey]; !exists {
		s.consumerMu.RUnlock()
		return nil, status.Error(codes.NotFound, "invalid public key")
	}
	s.consumerMu.RUnlock()

	s.playerDataMu.Lock()
	defer s.playerDataMu.Unlock()

	const maxAttempt = 10
	for i := 0; i < maxAttempt; i++ {
		id := uuid.New().String()
		if _, exists := s.clients[id]; !exists {
			s.clients[id] = &models.Client{Id: id}
			return &ExchangeReply{ClientId: id}, nil
		}
	}

	return nil, status.Error(codes.Internal, "failed to exchange")
}
