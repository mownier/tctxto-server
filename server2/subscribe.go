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

	rematchUpdates := s.getRematchInitialUpdates(player.Id)
	if len(rematchUpdates) > 0 {
		return append(updates, rematchUpdates...)
	}

	gameUpdates := s.getGameInitialUpdates(player)
	if len(gameUpdates) > 0 {
		return append(updates, gameUpdates...)
	}

	lobbyUpdates := s.getLobbyInitialUpdates(player.Id)
	if len(lobbyUpdates) > 0 {
		return append(updates, lobbyUpdates...)
	}

	return append(updates, s.createNavigationUpdate(NavigationPath_HOME))
}

func (s *Server) cleanupClientResources(clientId string) {
	s.clientSignal.delete(clientId)
}

func (s *Server) getLobbyInitialUpdates(playerId string) []*ServerUpdate {
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

func (s *Server) getGameInitialUpdates(you *models.Player) []*ServerUpdate {
	gameId, ok := s.playerGame.get(you.Id)
	if !ok {
		return []*ServerUpdate{}
	}

	game, ok := s.games.get(gameId)
	if !ok {
		s.playerGame.delete(you.Id)
		return []*ServerUpdate{}
	}

	updates := []*ServerUpdate{
		s.createNavigationUpdate(NavigationPath_GAME),
		s.createGameStartUpdate(game, you),
		s.createNextMoverUpdate(s.areYouTheMover(game, you)),
	}
	updates = append(updates, s.createMoveUpdates(game)...)

	switch game.Result {
	case models.GameResult_DRAW:
		updates = append(updates, s.createDrawUpdate())
	case models.GameResult_WIN:
		updates = append(updates, s.createWinnerUpdate(s.areYouTheMover(game, you), Technicality_NO_PROBLEM))
	case models.GameResult_WIN_BY_FORFEIT:
		updates = append(updates, s.createWinnerUpdate(s.areYouTheMover(game, you), Technicality_BY_FORFEIT))
	}

	return updates
}

func (s *Server) getRematchInitialUpdates(playerId string) []*ServerUpdate {
	rematchId, exists := s.playerRematch.get(playerId)
	if !exists {
		return []*ServerUpdate{}
	}

	rematch, exists := s.rematches.get(rematchId)
	if !exists {
		s.playerRematch.delete(playerId)
		return []*ServerUpdate{}
	}

	pd, exists := rematch.GetPlayerDecision(playerId)
	if !exists {
		for _, pd := range rematch.PlayerDecisions {
			if pd.Player.Id == playerId {
				s.playerRematch.delete(pd.Player.Id)
				break
			}
		}
		s.rematches.delete(rematch.Id)
		return []*ServerUpdate{}
	}

	if pd.Decision == models.Decision_UNDECIDED || pd.Decision == models.Decision_NO {
		return []*ServerUpdate{}
	}

	return []*ServerUpdate{s.createNavigationUpdate(NavigationPath_REMATCH)}
}
