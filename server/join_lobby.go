package server

import (
	context "context"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
)

func (s *Server) JoinLobby(ctx context.Context, in *JoinLobbyRequest) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "join lobby was cancelled")
	if err != nil {
		return nil, err
	}
	return s.joinLobbyInternal(clientId, in.LobbyId)
}

func (s *Server) joinLobbyInternal(clientId, lobbyId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createJoinLobbyReply(outcome)})
		return &Empty{}, nil
	}

	_, exists := s.playerLobbyMap[player.Id]
	if exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createJoinLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player has already a lobby",
		})})
		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]
	if !exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createJoinLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby not found",
		})})
		return &Empty{}, nil
	}

	if _, exists := lobby.Players[player.Id]; !exists {
		lobby.Players[player.Id] = player
	}

	updates := []*SubscriptionUpdate{s.createJoinLobbyReply(&Outcome{Ok: true})}
	if lobby != nil {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_MY_LOBBY), s.createMyLobbyDetails(lobby))
		s.notifyLobbyMembersOnJoin(clientId, lobby)
	}
	s.queueUpdatesAndSignal(clientId, updates)

	return &Empty{}, nil
}

func (s *Server) notifyLobbyMembersOnJoin(joiningClientId string, lobby *models.Lobby) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, player := range lobby.Players {
		if player == nil || s.playerClientMap[player.Id] == joiningClientId {
			continue
		}

		otherClientId, exists := s.playerClientMap[player.Id]
		if !exists {
			continue
		}

		s.queueUpdatesAndSignal(otherClientId, []*SubscriptionUpdate{s.createMyLobbyJoinerUpdate(player)})
	}
}
