package server2

import (
	"strings"
	"txtcto/models"
)

func (s *Server) searchLobby(clientId string, in *LobbySearchRequest) error {
	_, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createLobbySearchReply(outcome))
		return nil
	}

	const listLimit = 20
	list := []*models.Lobby{}
	s.lobbies.forEach(func(key string, lobby *models.Lobby) bool {
		if strings.Contains(strings.ToLower(lobby.Name), strings.ToLower(in.Name)) {
			list = append(list, lobby)
		}
		return len(list) <= listLimit
	})

	s.queueServerUpdatesAndSignal(clientId,
		s.createLobbySearchReply(&Outcome{Ok: true}),
		s.createLobbySearchResult(list),
	)

	return nil
}
