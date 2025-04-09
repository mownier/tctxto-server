package server

import (
	context "context"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) JoinLobby(ctx context.Context, in *JoinLobbyRequest) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "join lobby was cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.joinLobby(clientId, in.LobbyId)
	}
}

func (s *Server) joinLobby(clientId, lobbyId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	player, outcome := s.checkPlayer(clientId)

	if !outcome.Ok {
		s.queueSignalUpdatesOnJoinLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	_, exists := s.playerLobbyMap[player.Id]

	if exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "player has already a lobby"

		s.queueSignalUpdatesOnJoinLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "lobby not found"

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

		s.mu.Lock()
		defer s.mu.Unlock()

		for _, player := range lobby.Players {
			if player == nil {
				continue
			}

			otherClientIdMap, exists := s.playerClientMap[player.Id]

			if !exists {
				continue
			}

			for otherClientId := range otherClientIdMap {
				if otherClientId == clientId {
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
