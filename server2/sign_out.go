package server2

import (
	codes "google.golang.org/grpc/codes"
)

func (s *Server) signOut(clientId string) error {
	playerId, exists := s.clientPlayer.get(clientId)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignOutReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player not found",
		}))
		return nil
	}

	if _, exists := s.players.get(playerId); !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignOutReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player details not found",
		}))
		return nil
	}

	if _, exists := s.playerGame.get(playerId); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignOutReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player is currently in a game",
		}))
		return nil
	}

	s.clientPlayer.delete(clientId)
	s.playerClient.delete(playerId)

	s.queueServerUpdatesAndSignal(clientId,
		s.createSignOutReply(&Outcome{Ok: true}),
		s.createNavigationUpdate(NavigationPath_WELCOME, true),
	)

	return nil
}
