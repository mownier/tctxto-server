package server2

func (s *Server) changePlayerDisplayName(clientId string, in *ChangePlayerDisplayNameRequest) error {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createChangePlayerDisplayNameReply(outcome))
		return nil
	}

	if player.DisplayName == in.DisplayName {
		s.queueServerUpdatesAndSignal(clientId, s.createChangePlayerDisplayNameReply(&Outcome{Ok: true}))
		return nil
	}

	player.DisplayName = in.DisplayName

	s.queueServerUpdatesAndSignal(clientId,
		s.createChangePlayerDisplayNameReply(&Outcome{Ok: true}),
		s.createPlayerDisplayNameUpdate(in.DisplayName),
	)

	if lobbyId, exists := s.playerLobby.get(player.Id); exists {
		if lobby, exists := s.lobbies.get(lobbyId); exists {
			for _, lobbyPlayer := range lobby.Players {
				if lobbyPlayerClientId, exists := s.playerClient.get(lobbyPlayer.Id); exists {
					s.queueServerUpdatesAndSignal(lobbyPlayerClientId, s.createMyLobbyDetails(lobby))
				}
			}
		}
	}

	return nil
}
