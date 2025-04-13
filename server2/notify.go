package server2

import (
	"context"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Notify(ctx context.Context, update *ClientUpdate) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "notify was cancelled")
	if err != nil {
		return nil, err
	}

	if _, exists := s.clients.get(clientId); !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	switch update := update.Type.(type) {
	case *ClientUpdate_SignUpRequest:
		err = s.signUp(clientId, update.SignUpRequest)
	case *ClientUpdate_SignOutRequest:
		err = s.signOut(clientId)
	case *ClientUpdate_SignInRequest:
		err = s.signIn(clientId, update.SignInRequest)
	}

	if err != nil {
		return nil, err
	}

	return &Empty{}, nil
}
