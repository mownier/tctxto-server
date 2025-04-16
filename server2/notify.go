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
	case *ClientUpdate_CreateLobbyRequest:
		err = s.createLobby(clientId, update.CreateLobbyRequest)
	case *ClientUpdate_JoinLobbyRequest:
		err = s.joinLobby(clientId, update.JoinLobbyRequest)
	case *ClientUpdate_LeaveMyLobbyRequest:
		err = s.leaveMyLobby(clientId)
	case *ClientUpdate_CreateGameRequest:
		err = s.createGame(clientId, update.CreateGameRequest)
	case *ClientUpdate_MakeMoveRequest:
		err = s.makeMove(clientId, update.MakeMoveRequest)
	case *ClientUpdate_RematchRequest:
		err = s.rematch(clientId, update.RematchRequest)
	}

	if err != nil {
		return nil, err
	}

	return &Empty{}, nil
}
