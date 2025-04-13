package server2

import (
	"time"
	"txtcto/models"

	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Subscribe(in *SubscribeRequest, stream TicTacToe_SubscribeServer) error {
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

	s.sendInitialServerUpdates(clientId, in.Action, stream)

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
		switch update := serverUpdate.Type.(type) {
		case *ServerUpdate_NavigationUpdate:
			s.clientLastNavigationPath.set(clientId, update.NavigationUpdate.Path)
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

func (s *Server) sendInitialServerUpdates(clientId string, action SubscriptionAction, stream TicTacToe_SubscribeServer) {
	initialUpdates := s.initialServerUpdates(clientId, action)

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

func (s *Server) initialServerUpdates(clientId string, action SubscriptionAction) []*ServerUpdate {
	playerId, playerExists := s.clientPlayer.get(clientId)
	if !playerExists || !s.isPlayerActive(playerId) {
		lastPath, exists := s.clientLastNavigationPath.get(clientId)
		var refresh bool
		if !exists || lastPath != NavigationPath_WELCOME || action == SubscriptionAction_INITIAL {
			refresh = true
		} else {
			refresh = false
		}
		return []*ServerUpdate{s.createNavigationUpdate(NavigationPath_WELCOME, refresh)}
	}

	gameUpdates := s.getGameInitialUpdates(clientId, playerId, action)
	if len(gameUpdates) > 0 {
		return gameUpdates
	}

	lobbyUpdates := s.getLobbyInitialUpdates(clientId, playerId, action)
	if len(lobbyUpdates) > 0 {
		return lobbyUpdates
	}

	lastPath, exists := s.clientLastNavigationPath.get(clientId)
	var refresh bool
	if !exists || lastPath != NavigationPath_HOME || action == SubscriptionAction_INITIAL {
		refresh = true
	} else {
		refresh = false
	}

	return []*ServerUpdate{s.createNavigationUpdate(NavigationPath_HOME, refresh)}
}

func (s *Server) isPlayerActive(playerId string) bool {
	_, ok := s.players.get(playerId)
	return ok
}

func (s *Server) cleanupClientResources(clientId string) {
	s.clientSignal.delete(clientId)
}

func (s *Server) getLobbyInitialUpdates(clientId, playerId string, action SubscriptionAction) []*ServerUpdate {
	lobbyId, ok := s.playerLobby.get(playerId)
	if ok {
		if lobby, ok := s.lobbies.get(lobbyId); ok {
			updates := []*ServerUpdate{}

			lastPath, exists := s.clientLastNavigationPath.get(clientId)
			var refresh bool
			if !exists || lastPath != NavigationPath_MY_LOBBY || action == SubscriptionAction_INITIAL {
				refresh = true
			} else {
				refresh = false
			}

			updates = append(updates, s.createNavigationUpdate(NavigationPath_MY_LOBBY, refresh))

			lastLobby, exists := s.clientLastLobby.get(clientId)
			if !exists || !models.DeepCompareLobby(lobby, lastLobby) || action == SubscriptionAction_INITIAL {
				refresh = true
			} else {
				refresh = false
			}
			if refresh {
				s.clientLastLobby.set(clientId, lobby.DeepCopy())
				updates = append(updates, s.createMyLobbyDetails(lobby))
			}

			return updates
		} else {
			s.playerLobby.delete(playerId)
		}
	}
	return nil
}

func (s *Server) getGameInitialUpdates(clientId, playerId string, action SubscriptionAction) []*ServerUpdate {
	gameId, ok := s.playerGame.get(playerId)
	if ok {
		if game, ok := s.games.get(gameId); ok {
			if game.Result < models.GameResult_DRAW {
				updates := []*ServerUpdate{}

				lastPath, exists := s.clientLastNavigationPath.get(clientId)
				var refresh bool
				if !exists || lastPath != NavigationPath_GAME || action == SubscriptionAction_INITIAL {
					refresh = true
				} else {
					refresh = false
				}

				updates = append(updates, s.createNavigationUpdate(NavigationPath_GAME, refresh))

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
