package server2

import codes "google.golang.org/grpc/codes"

func (s *Server) signIn(clientId string, in *SignInRequest) error {
	playerId, exists := s.playerNameId.get(in.Name)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignInReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player with name not found",
		}))
		return nil
	}

	player, exists := s.players.get(playerId)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createSignInReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player not found",
		}))
		return nil
	}

	if player.Pass != in.Pass {
		s.queueServerUpdatesAndSignal(clientId, s.createSignInReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.PermissionDenied),
			ErrorMessage: "player credentials not valid",
		}))
		return nil
	}

	if oldClientId, exists := s.playerClient.get(player.Id); exists {
		if oldClientId != clientId {
			s.clientPlayer.delete(oldClientId)

			s.queueServerUpdatesAndSignal(oldClientId,
				s.createNavigationUpdate(NavigationPath_WELCOME, true),
				s.createPlayerClientUpdate("You are using another client"),
			)
		}
	}

	s.clientPlayer.set(clientId, player.Id)
	s.playerClient.set(player.Id, clientId)

	updates := []*ServerUpdate{
		s.createSignInReply(&Outcome{Ok: true}),
		s.createPlayerDisplayNameUpdate(player.DisplayName),
	}
	updates = append(updates, s.initialServerUpdates(clientId, SubscriptionAction_INITIAL)...)

	s.queueServerUpdatesAndSignal(clientId, updates...)

	return nil
}
