package server2

import (
	"google.golang.org/grpc/codes"
)

func (s *Server) joinLobby(clientId string, in *JoinLobbyRequest) error {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createJoinLobbyReply(outcome))
		return nil
	}

	if _, exists := s.playerLobby.get(player.Id); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createJoinLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player has already in a lobby",
		}))
	}

	lobby, exists := s.lobbies.get(in.LobbyId)
	if !exists {
		s.queueServerUpdatesAndSignal(clientId, s.createJoinLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "lobby does not exists",
		}))
		return nil
	}

	lobby.Players[player.Id] = player

	s.playerLobby.set(player.Id, lobby.Id)

	for _, member := range lobby.Players {
		if member.Id == player.Id {
			continue
		}
		if memberClientId, exists := s.playerClient.get(member.Id); exists {
			s.queueServerUpdatesAndSignal(memberClientId,
				s.createMyLobbyJoinerUpdate(player.Id, player.DisplayName),
			)
		}
	}

	s.queueServerUpdatesAndSignal(clientId,
		s.createJoinLobbyReply(&Outcome{Ok: true}),
		s.createNavigationUpdate(NavigationPath_MY_LOBBY),
		s.createMyLobbyDetails(lobby),
	)

	return nil
}
