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
		s.queueSignalUpdatesOnJoinLobby(clientId, outcome, nil)
		return &Empty{}, nil
	}

	_, exists := s.playerLobbyMap[player.Id]
	if exists {
		outcome = &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player has already a lobby",
		}
		s.queueSignalUpdatesOnJoinLobby(clientId, outcome, nil)
		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]
	if !exists {
		outcome = &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby not found",
		}
		s.queueSignalUpdatesOnJoinLobby(clientId, outcome, nil)
		return &Empty{}, nil
	}

	if _, exists := lobby.Players[player.Id]; !exists {
		lobby.Players[player.Id] = player
	}

	s.queueSignalUpdatesOnJoinLobby(clientId, &Outcome{Ok: true}, lobby)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnJoinLobby(clientId string, outcome *Outcome, lobby *models.Lobby) {
	updates := []*SubscriptionUpdate{
		s.createJoinLobbyReply(outcome),
	}

	if lobby != nil {
		updates = append(updates,
			s.createNavigationUpdate(NavigationPath_MY_LOBBY),
			s.createMyLobbyDetails(lobby),
		)
		s.notifyLobbyMembersOnJoin(clientId, lobby)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}
	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], updates...)
	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:
			break
		default:
			break
		}
	}
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

		if _, exists := s.clientUpdatesMap[otherClientId]; !exists {
			s.clientUpdatesMap[otherClientId] = []*SubscriptionUpdate{}
		}

		s.clientUpdatesMap[otherClientId] = append(s.clientUpdatesMap[otherClientId],
			s.createMyLobbyJoinerUpdate(player),
		)

		if signal, exists := s.clientSignalMap[otherClientId]; exists {
			select {
			case signal <- struct{}{}:
				break
			default:
				break
			}
		}
	}
}
