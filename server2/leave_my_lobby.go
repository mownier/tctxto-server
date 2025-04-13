package server2

import "google.golang.org/grpc/codes"

func (s *Server) leaveMyLobby(clientId string) error {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createLeaveMyLobbyReply(outcome))
		return nil
	}

	lobbyId, exists := s.playerLobby.get(player.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createLeaveMyLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player does not belong to any lobby",
		}))
	}

	lobby, exists := s.lobbies.get(lobbyId)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createLeaveMyLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby does not exists",
		}))
		return nil
	}

	assignedId := lobby.PlayerAssignedId[player.Id]

	delete(lobby.Players, player.Id)
	delete(lobby.AssignedIds, assignedId)
	delete(lobby.PlayerAssignedId, player.Id)

	s.playerLobby.delete(player.Id)

	for _, member := range lobby.Players {
		if member.Id == player.Id {
			continue
		}
		if memberClientId, exists := s.playerClient.get(member.Id); exists {
			s.queueServerUpdatesAndSignal(memberClientId,
				s.createMyLobbyLeaverUpdate(assignedId, player.Name),
			)
		}
	}

	s.queueServerUpdatesAndSignal(clientId,
		s.createLeaveMyLobbyReply(&Outcome{Ok: true}),
		s.createNavigationUpdate(NavigationPath_HOME),
	)

	return nil
}
