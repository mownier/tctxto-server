package server2

import (
	"time"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Subscribe(emp *Empty, stream TicTacToe_SubscribeServer) error {
	publicKey, err := s.extractPublicKeyWithCancel(stream.Context(), "subscribe was cancelled")
	if err != nil {
		return err
	}

	if _, exists := s.consumers.get(publicKey); !exists {
		return status.Error(codes.PermissionDenied, "rejected")
	}

	clientId, err := s.extractClientId(stream.Context())
	if err != nil {
		clientId = uuid.New().String()
		s.clients.set(clientId, &models.Client{Id: clientId})
	}

	_, exists := s.clients.get(clientId)
	if !exists {
		return status.Error(codes.NotFound, "unknown client")
	}

	if err := stream.Send(s.createClientAssignmentUpdate(clientId)); err != nil {
		return status.Error(codes.Internal, "unable to send client assignment update")
	}

	if _, exists := s.clientServerUpdates.get(clientId); !exists {
		s.clientServerUpdates.set(clientId, []*ServerUpdate{})
	}

	if _, exists := s.clientSignal.get(clientId); !exists {
		s.clientSignal.set(clientId, make(chan struct{}, 1))
	}

	s.sendInitialServerUpdates(clientId, stream)

	signal, _ := s.clientSignal.get(clientId)

	defer s.cleanupClientResources(clientId)

	pingInterval := 100 * time.Millisecond
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "subscribe was done")
		case <-pingTicker.C:
			if err := stream.Send(s.createPing()); err != nil {
				return err
			}
		case <-signal:
			if err := s.sendServerUpdates(stream, clientId); err != nil {
				return err
			}
		}
	}
}

func (s *Server) sendServerUpdates(stream TicTacToe_SubscribeServer, clientId string) error {
	lastIndex, exists := s.clientLastIndexServerUpdate.get(clientId)
	startIndex := 0
	if exists && lastIndex != -1 {
		startIndex = lastIndex + 1
	}

	if _, exists := s.clientServerUpdates.get(clientId); !exists {
		s.clientServerUpdates.set(clientId, []*ServerUpdate{})
	}
	serverUpdates, _ := s.clientServerUpdates.get(clientId)

	var serverUpdatesToSend []*ServerUpdate

	if startIndex < len(serverUpdates) {
		serverUpdatesToSend = serverUpdates[startIndex:]
	} else {
		serverUpdatesToSend = []*ServerUpdate{}
	}

	sentCount := 0
	var err error

	for _, serverUpdate := range serverUpdatesToSend {
		if e := stream.Send(serverUpdate); e != nil {
			err = e
			break
		}
		sentCount++
	}

	if sentCount > 0 {
		index, exists := s.clientLastIndexServerUpdate.get(clientId)
		if !exists {
			index = -1
		}
		index += sentCount
		s.clientLastIndexServerUpdate.set(clientId, index)
	}

	return err
}

func (s *Server) sendInitialServerUpdates(clientId string, stream TicTacToe_SubscribeServer) {
	initialUpdates := s.initialServerUpdates(clientId)

	serverUpdates, exists := s.clientServerUpdates.get(clientId)
	if !exists {
		serverUpdates = []*ServerUpdate{}
	}

	if playerId, exists := s.clientPlayer.get(clientId); exists {
		if player, exists := s.players.get(playerId); exists {
			serverUpdates = append(serverUpdates, s.createPlayerDisplayNameUpdate(player.DisplayName))
		}
	}

	serverUpdates = append(serverUpdates, initialUpdates...)
	s.clientServerUpdates.set(clientId, serverUpdates)

	s.sendServerUpdates(stream, clientId)
}

func (s *Server) initialServerUpdates(clientId string) []*ServerUpdate {
	player, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		return []*ServerUpdate{s.createNavigationUpdate(NavigationPath_WELCOME)}
	}

	updates := []*ServerUpdate{s.createPlayerDisplayNameUpdate(player.DisplayName)}

	gameUpdates := s.getGameInitialUpdates(clientId, player.Id)
	if len(gameUpdates) > 0 {
		return append(updates, gameUpdates...)
	}

	lobbyUpdates := s.getLobbyInitialUpdates(clientId, player.Id)
	if len(lobbyUpdates) > 0 {
		return append(updates, lobbyUpdates...)
	}

	return append(updates, s.createNavigationUpdate(NavigationPath_HOME))
}

func (s *Server) cleanupClientResources(clientId string) {
	s.clientSignal.delete(clientId)
}

func (s *Server) getLobbyInitialUpdates(clientId, playerId string) []*ServerUpdate {
	lobbyId, ok := s.playerLobby.get(playerId)
	if ok {
		if lobby, ok := s.lobbies.get(lobbyId); ok {
			return []*ServerUpdate{
				s.createNavigationUpdate(NavigationPath_MY_LOBBY),
				s.createMyLobbyDetails(lobby),
			}
		} else {
			s.playerLobby.delete(playerId)
		}
	}
	return nil
}

func (s *Server) getGameInitialUpdates(clientId, playerId string) []*ServerUpdate {
	gameId, ok := s.playerGame.get(playerId)
	if ok {
		if game, ok := s.games.get(gameId); ok {
			if game.Result < models.GameResult_DRAW {
				updates := []*ServerUpdate{}

				updates = append(updates, s.createNavigationUpdate(NavigationPath_GAME))

				you, other := s.getGamePlayers(game, playerId)

				updates = append(updates,
					s.createGameStartUpdate(game, you, other),
					s.createNextMoverUpdate(game),
				)

				moveUpdates := s.createMoveUpdates(game)
				updates = append(updates, moveUpdates...)

				return updates
			} else {
				s.playerGame.delete(playerId)
			}
		} else {
			s.playerGame.delete(playerId)
		}
	}

	return nil
}

func (s *Server) getGamePlayers(game *models.Game, playerId string) (*models.Player, *models.Player) {
	var you *models.Player
	var other *models.Player
	if game.MoverX.Id == playerId {
		you = game.MoverX
		other = game.MoverO
	} else if game.MoverO.Id == playerId {
		you = game.MoverO
		other = game.MoverX
	}
	return you, other
}
