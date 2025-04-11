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
	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createLeaveMyLobbyReply(outcome)})
		return &Empty{}, nil
	}

	s.lobbyGameMu.Lock()

	lobbyId, exists := s.playerLobbyMap[player.Id]
	if !exists {
		s.lobbyGameMu.Unlock()
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createLeaveMyLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.Internal),
			ErrorMessage: "player does not have a lobby",
		})})
		return &Empty{}, nil
	}

	lobby, exists := s.lobbies[lobbyId]
	if !exists {
		s.lobbyGameMu.Unlock()
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createLeaveMyLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby not found",
		})})
		return &Empty{}, nil
	}

	if _, exists := lobby.Players[player.Id]; !exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createLeaveMyLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player is not a lobby member",
		})})
		return &Empty{}, nil
	}

	leavingPlayer := lobby.Players[player.Id]

	s.lobbyGameMu.Unlock()

	s.cleanupLeavingLobbyMember(lobby, player.Id)

	updates := []*SubscriptionUpdate{s.createLeaveMyLobbyReply(&Outcome{Ok: true})}
	if leavingPlayer != nil {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_HOME))
		s.notifyLobbyMembersOnLeave(clientId, leavingPlayer)
	} else {
		updates = append(updates, s.createNavigationUpdate(NavigationPath_HOME))
	}
	s.queueUpdatesAndSignal(clientId, updates)

	return &Empty{}, nil
}

func (s *Server) notifyLobbyMembersOnLeave(leavingClientId string, leavingPlayer *models.Player) {
	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	if lobbyId, exists := s.playerLobbyMap[leavingPlayer.Id]; exists {
		if lobby, exists := s.lobbies[lobbyId]; exists {
			for _, p := range lobby.Players {
				s.playerDataMu.Lock()
				defer s.playerDataMu.Unlock()

				if p == nil || s.playerClientMap[p.Id] == leavingClientId {
					continue
				}
				if otherClientId, exists := s.playerClientMap[p.Id]; exists {
					s.queueUpdatesAndSignal(otherClientId, []*SubscriptionUpdate{s.createMyLobbyLeaverUpdate(leavingPlayer)})
				}
			}
		}
	}
}

func (s *Server) cleanupLeavingLobbyMember(lobby *models.Lobby, playerId string) {
	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	delete(s.playerLobbyMap, playerId)
	delete(lobby.Players, playerId)
}
