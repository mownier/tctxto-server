package server

import (
	"log"
	"time"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) Subscribe(emp *Empty, stream TicTacToe_SubscribeServer) error {
	clientId, err := s.extractClientId(stream.Context())
	if err != nil {
		log.Printf("Subscribe client id = %s, extractClientId error = %v\n", clientId, err)
		return err
	}

	if s.hasActiveSubscription(clientId) {
		log.Printf("Subscribe client id  %s already has an active subscription. Rejecting new attempt.\n", clientId)
		return status.Error(codes.AlreadyExists, "client already subscribed")
		// Or you might want to handle this differently, like closing the old one.
	}

	log.Printf("Subscribe client id = %s, register activeSubscriptionsMu LOCK\n", clientId)
	s.activeSubscriptionsMu.Lock()
	log.Printf("Subscribe client id = %s, register activeSubscriptionsMu LOCKED\n", clientId)
	s.activeSubscriptions[clientId] = true
	s.activeSubscriptionsMu.Unlock()
	log.Printf("Subscribe client id = %s, register activeSubscriptionsMu UNLOCK\n", clientId)

	log.Printf("Subscribe client id = %s, isClientKnown START\n", clientId)
	if !s.isClientKnown(clientId) {
		log.Printf("Subscribe client id = %s, isClientKnown END, client not known\n", clientId)
		return status.Error(codes.NotFound, "unknown client")
	}
	log.Printf("Subscribe client id = %s, isClientKnown END, client known\n", clientId)

	log.Printf("Subscribe client id = %s, setupClientSubscriptionChannels START\n", clientId)
	s.setupClientSubscriptionChannels(clientId)
	log.Printf("Subscribe client id = %s, setupClientSubscriptionChannels END\n", clientId)

	log.Printf("Subscribe client id = %s, sendInitialClientUpdates START\n", clientId)
	s.sendInitialClientUpdates(clientId, stream)
	log.Printf("Subscribe client id = %s, sendInitialClientUpdates END\n", clientId)

	signal := s.getClientSignalChannel(clientId)
	log.Printf("Subscribe client id = %s, getClientSignalChannel END\n", clientId)

	defer func() {
		log.Printf("Subscribe client id = %s, unregister activeSubscriptionsMu LOCK\n", clientId)
		s.activeSubscriptionsMu.Lock()
		log.Printf("Subscribe client id = %s, unregister activeSubscriptionsMu LOCKED\n", clientId)
		delete(s.activeSubscriptions, clientId)
		s.activeSubscriptionsMu.Unlock()
		log.Printf("Subscribe client id = %s, unregister activeSubscriptionsMu UNLOCK\n", clientId)

		log.Printf("Subscribe client id = %s, cleanupClientSubscription START\n", clientId)
		s.cleanupClientSubscription(clientId)
		log.Printf("Subscribe client id = %s, cleanupClientSubscription END\n", clientId)
	}()

	pingInterval := 100 * time.Millisecond
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Subscribe client id = %s, cancelled\n", clientId)
			return status.Error(codes.Canceled, "subscribe was cancelled")
		case <-signal:
			log.Printf("Subscribe client id = %s, signal received, sendClientUpdates START\n", clientId)
			if err := s.sendClientUpdates(clientId, stream); err != nil {
				log.Printf("Subscribe client id = %s, sendClientUpdates error = %v\n", clientId, err)
				return err
			}
			log.Printf("Subscribe client id = %s, sendClientUpdates END\n", clientId)
		case <-pingTicker.C:
			//log.Printf("Subscribe client id = %s, sending ping START\n", clientId)
			ping := s.createPing()
			if err := stream.Send(ping); err != nil {
				//log.Printf("Subscribe client id = %s, error sending ping: %v\n", clientId, err)
				return err
			}
			//log.Printf("Subscribe client id = %s, sending ping END\n", clientId)
		}
	}
}

func (s *Server) hasActiveSubscription(clientId string) bool {
	s.activeSubscriptionsMu.RLock()
	defer s.activeSubscriptionsMu.RUnlock()
	return s.activeSubscriptions[clientId]
}

func (s *Server) isClientKnown(clientId string) bool {
	s.playerDataMu.RLock()
	defer s.playerDataMu.RUnlock()
	_, exists := s.clients[clientId]
	return exists
}

func (s *Server) setupClientSubscriptionChannels(clientId string) {
	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()
	if _, exists := s.clientSignalMap[clientId]; !exists {
		s.clientSignalMap[clientId] = make(chan struct{}, 1)
	}
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}
}

func (s *Server) getClientSignalChannel(clientId string) <-chan struct{} {
	s.clientSubscriptionMu.RLock()
	defer s.clientSubscriptionMu.RUnlock()
	return s.clientSignalMap[clientId]
}

func (s *Server) sendInitialClientUpdates(clientId string, stream TicTacToe_SubscribeServer) {
	log.Printf("sendInitialClientUpdates client id = %s, clientInitialUpdates START\n", clientId)
	initialUpdates := s.clientInitialUpdates(clientId)
	log.Printf("sendInitialClientUpdates client id = %s, clientInitialUpdates END, updates count = %d\n", clientId, len(initialUpdates))

	log.Printf("sendInitialClientUpdates client id = %s, s.clientSubscriptionMu LOCK\n", clientId)
	s.clientSubscriptionMu.Lock()
	log.Printf("sendInitialClientUpdates client id = %s, s.clientSubscriptionMu LOCKED\n", clientId)
	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], initialUpdates...)
	s.clientSubscriptionMu.Unlock()
	log.Printf("sendInitialClientUpdates client id = %s, s.clientSubscriptionMu UNLOCK\n", clientId)

	log.Printf("sendInitialClientUpdates client id = %s, sendClientUpdates START\n", clientId)
	s.sendClientUpdates(clientId, stream)
	log.Printf("sendInitialClientUpdates client id = %s, sendClientUpdates END\n", clientId)
}

func (s *Server) sendClientUpdates(clientId string, stream TicTacToe_SubscribeServer) error {
	log.Printf("sendClientUpdates client id = %s, s.clientSubscriptionMu LOCK\n", clientId)
	s.clientSubscriptionMu.Lock()
	log.Printf("sendClientUpdates client id = %s, s.clientSubscriptionMu LOCKED\n", clientId)
	defer func() {
		s.clientSubscriptionMu.Unlock()
		log.Printf("sendClientUpdates client id = %s, s.clientSubscriptionMu UNLOCK\n", clientId)
	}()

	log.Printf("sendClientUpdates client id = %s, accessing clientLastIndexUpdate\n", clientId)
	lastIndex, exists := s.clientLastIndexUpdate[clientId]
	log.Printf("sendClientUpdates client id = %s, clientLastIndexUpdate result: lastIndex=%d, exists=%v\n", clientId, lastIndex, exists)
	startIndex := 0
	if exists && lastIndex != -1 {
		startIndex = lastIndex + 1
	}
	log.Printf("sendClientUpdates client id = %s, startIndex = %d\n", clientId, startIndex)

	log.Printf("sendClientUpdates client id = %s, accessing clientUpdatesMap\n", clientId)
	clientUpdates := s.clientUpdatesMap[clientId]
	var updatesToSend []*SubscriptionUpdate

	if clientUpdates != nil && startIndex < len(clientUpdates) {
		updatesToSend = clientUpdates[startIndex:]
		log.Printf("sendClientUpdates client id = %s, updatesToSend count = %d (from index %d)\n", clientId, len(updatesToSend), startIndex)
	} else {
		updatesToSend = []*SubscriptionUpdate{} // Or nil, depending on your logic
		if clientUpdates == nil {
			log.Printf("sendClientUpdates client id = %s, clientUpdatesMap[%s] is nil\n", clientId, clientId)
		} else {
			log.Printf("sendClientUpdates client id = %s, startIndex (%d) is out of bounds (length: %d)\n", clientId, startIndex, len(clientUpdates))
		}
	}

	sentCount := 0
	var err error

	for i, update := range updatesToSend {
		log.Printf("sendClientUpdates client id = %s, sending update %d: %+v\n", clientId, i, update)
		if e := stream.Send(update); e != nil {
			log.Printf("sendClientUpdates client id = %s, stream.Send error = %v\n", clientId, e)
			err = e
			break
		}
		switch data := update.Data.GetSubscriptionUpdateDataType().(type) {
		case *SubscriptionUpdateData_NavigationUpdate:
			s.clientLastNavPathUpdateMu.Lock()
			s.clientLastNavPathUpdate[clientId] = data.NavigationUpdate.Path
			s.clientLastNavPathUpdateMu.Unlock()
		}
		sentCount++
		log.Printf("sendClientUpdates client id = %s, update %d sent\n", clientId, i)
	}

	if sentCount > 0 {
		s.clientLastIndexUpdate[clientId] += sentCount
		log.Printf("sendClientUpdates client id = %s, clientLastIndexUpdate updated to %d\n", clientId, s.clientLastIndexUpdate[clientId])
	}

	log.Printf("sendClientUpdates client id = %s, returning error = %v\n", clientId, err)
	return err
}

func (s *Server) clientInitialUpdates(clientId string) []*SubscriptionUpdate {
	log.Printf("clientInitialUpdates client id = %s, s.playerDataMu LOCK\n", clientId)
	s.playerDataMu.RLock()
	log.Printf("clientInitialUpdates client id = %s, s.playerDataMu LOCKED\n", clientId)

	log.Printf("clientInitialUpdates client id = %s, checking clientPlayerMap\n", clientId)
	playerId, playerExists := s.clientPlayerMap[clientId]
	log.Printf("clientInitialUpdates client id = %s, clientPlayerMap result: playerId=%s, exists=%v\n", clientId, playerId, playerExists)

	s.playerDataMu.RUnlock()
	log.Printf("clientInitialUpdates client id = %s, s.playerDataMu UNLOCK\n", clientId)

	if !playerExists || !s.isPlayerActive(playerId) {
		s.clientLastNavPathUpdateMu.Lock()
		lastPath, exists := s.clientLastNavPathUpdate[clientId]
		s.clientLastNavPathUpdateMu.Unlock()

		if !exists || lastPath != NavigationPath_LOGIN {
			return []*SubscriptionUpdate{s.createNavigationUpdate(NavigationPath_LOGIN)}
		}

		return []*SubscriptionUpdate{}
	}

	log.Printf("clientInitialUpdates client id = %s, calling getGameInitialUpdates\n", clientId)
	gameUpdates := s.getGameInitialUpdates(clientId, playerId)
	if len(gameUpdates) > 0 {
		return gameUpdates
	}

	log.Printf("clientInitialUpdates client id = %s, calling getLobbyInitialUpdates\n", clientId)
	lobbyUpdates := s.getLobbyInitialUpdates(clientId, playerId)
	log.Printf("clientInitialUpdates client id = %s, getLobbyInitialUpdates returned %d updates\n", clientId, len(lobbyUpdates))
	if len(lobbyUpdates) > 0 {
		return lobbyUpdates
	}

	s.clientLastNavPathUpdateMu.Lock()
	lastPath, exists := s.clientLastNavPathUpdate[clientId]
	s.clientLastNavPathUpdateMu.Unlock()

	if !exists || lastPath != NavigationPath_HOME {
		log.Printf("clientInitialUpdates client id = %s, returning HOME update\n", clientId)
		return []*SubscriptionUpdate{s.createNavigationUpdate(NavigationPath_HOME)}
	}

	log.Printf("clientInitialUpdates client id = %s, returning empty\n", clientId)
	return []*SubscriptionUpdate{}
}

func (s *Server) isPlayerActive(playerId string) bool {
	log.Printf("isPlayerActive playerId = %s, s.playerDataMu RLOCK\n", playerId)
	s.playerDataMu.RLock()
	log.Printf("isPlayerActive playerId = %s, s.playerDataMu RLOCKED\n", playerId)
	defer func() {
		s.playerDataMu.RUnlock()
		log.Printf("isPlayerActive playerId = %s, s.playerDataMu RUNLOCK\n", playerId)
	}()
	_, ok := s.players[playerId]
	log.Printf("isPlayerActive playerId = %s, result = %v\n", playerId, ok)
	return ok
}

func (s *Server) getGameInitialUpdates(clientId, playerId string) []*SubscriptionUpdate {
	log.Printf("getGameInitialUpdates playerId = %s, s.lobbyGameMu LOCK\n", playerId)
	s.lobbyGameMu.Lock()
	log.Printf("getGameInitialUpdates playerId = %s, s.lobbyGameMu LOCKED\n", playerId)
	defer func() {
		s.lobbyGameMu.Unlock()
		log.Printf("getGameInitialUpdates playerId = %s, s.lobbyGameMu UNLOCK\n", playerId)
	}()

	log.Printf("getGameInitialUpdates playerId = %s, checking playerGameMap\n", playerId)
	gameId, ok := s.playerGameMap[playerId]
	log.Printf("getGameInitialUpdates playerId = %s, playerGameMap result: gameId=%s, exists=%v\n", playerId, gameId, ok)
	if ok {
		log.Printf("getGameInitialUpdates playerId = %s, checking games map\n", playerId)
		if game, ok := s.games[gameId]; ok {
			log.Printf("getGameInitialUpdates playerId = %s, found game: %+v\n", playerId, game)
			if game.Result < models.GameResult_DRAW {
				log.Printf("getGameInitialUpdates playerId = %s, game result is not DRAW\n", playerId)
				you, other := s.getGamePlayers(game, playerId)

				s.clientLastNavPathUpdateMu.Lock()
				lastPath, exists := s.clientLastNavPathUpdate[clientId]
				s.clientLastNavPathUpdateMu.Unlock()

				updates := []*SubscriptionUpdate{}

				if !exists || lastPath != NavigationPath_GAME {
					updates = append(updates, s.createNavigationUpdate(NavigationPath_GAME))
				}

				updates = append(updates,
					s.createGameStartUpdate(game, you, other),
					s.createNextMoverUpdate(game),
				)

				moveUpdates := s.createMoveUpdates(game)
				updates = append(updates, moveUpdates...)
				log.Printf("getGameInitialUpdates playerId = %s, returning %d game updates\n", playerId, len(updates))
				return updates
			} else {
				log.Printf("getGameInitialUpdates playerId = %s, game result is DRAW or greater, deleting from playerGameMap\n", playerId)
				delete(s.playerGameMap, playerId)
			}
		} else {
			log.Printf("getGameInitialUpdates playerId = %s, game not found in games map, deleting from playerGameMap\n", playerId)
			delete(s.playerGameMap, playerId)
		}
	}
	log.Printf("getGameInitialUpdates playerId = %s, returning nil\n", playerId)
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

func (s *Server) getLobbyInitialUpdates(clientId, playerId string) []*SubscriptionUpdate {
	log.Printf("getLobbyInitialUpdates playerId = %s, s.lobbyGameMu LOCK\n", playerId)
	s.lobbyGameMu.Lock()
	log.Printf("getLobbyInitialUpdates playerId = %s, s.lobbyGameMu LOCKED\n", playerId)
	defer func() {
		s.lobbyGameMu.Unlock()
		log.Printf("getLobbyInitialUpdates playerId = %s, s.lobbyGameMu UNLOCK\n", playerId)
	}()

	log.Printf("getLobbyInitialUpdates playerId = %s, checking playerLobbyMap\n", playerId)
	lobbyId, ok := s.playerLobbyMap[playerId]
	log.Printf("getLobbyInitialUpdates playerId = %s, playerLobbyMap result: lobbyId=%s, exists=%v\n", playerId, lobbyId, ok)
	if ok {
		log.Printf("getLobbyInitialUpdates playerId = %s, checking lobbies map\n", playerId)
		if lobby, ok := s.lobbies[lobbyId]; ok {
			log.Printf("getLobbyInitialUpdates playerId = %s, found lobby: %+v\n", playerId, lobby)

			s.clientLastNavPathUpdateMu.Lock()
			lastPath, exists := s.clientLastNavPathUpdate[clientId]
			s.clientLastNavPathUpdateMu.Unlock()

			updates := []*SubscriptionUpdate{}

			if !exists || lastPath != NavigationPath_MY_LOBBY {
				updates = append(updates, s.createNavigationUpdate(NavigationPath_MY_LOBBY))
			}

			updates = append(updates, s.createMyLobbyDetails(lobby))
			log.Printf("getLobbyInitialUpdates playerId = %s, returning %d lobby updates\n", playerId, len(updates))
			return updates
		} else {
			log.Printf("getLobbyInitialUpdates playerId = %s, lobby not found in lobbies map, deleting from playerLobbyMap\n", playerId)
			delete(s.playerLobbyMap, playerId)
		}
	}
	log.Printf("getLobbyInitialUpdates playerId = %s, returning nil\n", playerId)
	return nil
}

func (s *Server) cleanupClientSubscription(clientId string) {
	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()
	delete(s.clientSignalMap, clientId)
}
