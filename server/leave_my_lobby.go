package server

import (
	context "context"
	"txtcto/models"

	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) LeaveMyLobby(ctx context.Context) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "leave my lobby was cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.leaveMyLobby(clientId)
	}
}

func (s *Server) leaveMyLobby(clientId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	player, outcome := s.checkPlayer(clientId)

	if !outcome.Ok {
		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	lobbyId, exists := s.playerLobbyMap[player.Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.Internal)
		outcome.ErrorMessage = "player does not have a lobby"

		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "lobby not found"

		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	if _, exists := lobby.Players[player.Id]; exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "player is not a lobby member"

		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil)

		return &Empty{}, nil
	}

	s.cleanupLeavingLobbyMember(lobby, player.Id)
	s.queueSignalUpdatesOnLeaveMyLobby(clientId, &Outcome{Ok: true}, lobby)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnLeaveMyLobby(clientId string, outcome *Outcome, lobby *models.Lobby) {
	updates := []*SubscriptionUpdate{
		s.createLeaveMyLobbyReply(outcome),
	}

	if outcome.Ok {
		updates = append(updates,
			s.createNavigationUpdate(NavigationPath_HOME),
		)
	}

	if lobby != nil {
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
					s.createMyLobbyLeaverUpdate(player),
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

func (s *Server) cleanupLeavingLobbyMember(lobby *models.Lobby, playerId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.playerLobbyMap, playerId)
	delete(lobby.Players, playerId)
}
