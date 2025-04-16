package server2

import (
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func (s *Server) createLobby(clientId string, in *CreateLobbyRequest) error {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createCreateLobbyReply(outcome))
		return nil
	}

	if _, exists := s.playerLobby.get(player.Id); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createCreateLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player has already in a lobby",
		}))
	}

	lobbyId := uuid.New().String()
	if _, exists := s.lobbies.get(lobbyId); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createCreateLobbyReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.Internal),
			ErrorMessage: "unable to create lobby",
		}))
		return nil
	}

	lobby := &models.Lobby{
		Id:      lobbyId,
		Name:    in.Name,
		Creator: player,
		Players: make(map[string]*models.Player),
	}
	lobby.Players[player.Id] = player

	s.lobbies.set(lobby.Id, lobby)
	s.playerLobby.set(player.Id, lobby.Id)

	s.queueServerUpdatesAndSignal(clientId,
		s.createCreateLobbyReply(&Outcome{Ok: true}),
		s.createNavigationUpdate(NavigationPath_MY_LOBBY),
		s.createMyLobbyDetails(lobby),
	)

	return nil
}
