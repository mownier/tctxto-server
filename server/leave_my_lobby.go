package server

import (
	context "context"
	"txtcto/models"

	"google.golang.org/grpc/codes"
)

func (s *Server) LeaveMyLobby(ctx context.Context, emp *Empty) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "leave my lobby was cancelled")
	if err != nil {
		return nil, err
	}
	return s.leaveMyLobbyInternal(clientId)
}

func (s *Server) leaveMyLobbyInternal(clientId string) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil, nil)
		return &Empty{}, nil
	}

	lobbyId, exists := s.playerLobbyMap[player.Id]
	if !exists {
		outcome = &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.Internal),
			ErrorMessage: "player does not have a lobby",
		}
		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil, nil)
		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]
	if !exists {
		outcome = &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby not found",
		}
		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil, nil)
		return &Empty{}, nil
	}

	if _, exists := lobby.Players[player.Id]; !exists {
		outcome = &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player is not a lobby member",
		}
		s.queueSignalUpdatesOnLeaveMyLobby(clientId, outcome, nil, nil)
		return &Empty{}, nil
	}

	leavingPlayer := lobby.Players[player.Id]
	s.cleanupLeavingLobbyMember(lobby, player.Id)
	s.queueSignalUpdatesOnLeaveMyLobby(clientId, &Outcome{Ok: true}, lobby, leavingPlayer)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnLeaveMyLobby(clientId string, outcome *Outcome, lobby *models.Lobby, leavingPlayer *models.Player) {
	updates := []*SubscriptionUpdate{
		s.createLeaveMyLobbyReply(outcome),
	}

	if outcome.Ok {
		updates = append(updates,
			s.createNavigationUpdate(NavigationPath_HOME),
		)
	}

	if lobby != nil && leavingPlayer != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		for _, p := range lobby.Players {
			if p == nil || s.playerClientMap[p.Id] == clientId {
				continue
			}

			otherClientId, exists := s.playerClientMap[p.Id]
			if !exists {
				continue
			}

			if _, exists := s.clientUpdatesMap[otherClientId]; !exists {
				s.clientUpdatesMap[otherClientId] = []*SubscriptionUpdate{}
			}

			s.clientUpdatesMap[otherClientId] = append(s.clientUpdatesMap[otherClientId],
				s.createMyLobbyLeaverUpdate(leavingPlayer),
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
