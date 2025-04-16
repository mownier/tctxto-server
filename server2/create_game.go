package server2

import (
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func (s *Server) createGame(clientId string, in *CreateGameRequest) error {
	creator, outcome := s.validatePlayer(clientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createGameReply(outcome))
		return nil
	}

	player1ClientId, player1, outcome := s.getClientIdAndPlayer(in.Player1Id, "player 1")
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createGameReply(outcome))
		return nil
	}

	player2ClientId, player2, outcome := s.getClientIdAndPlayer(in.Player2Id, "player 2")
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(clientId, s.createGameReply(outcome))
		return nil
	}

	if player1.Id == player2.Id {
		s.queueServerUpdatesAndSignal(clientId, s.createGameReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "player 1 and player 2 is the same",
		}))
		return nil
	}

	gameId := uuid.New().String()

	if _, exists := s.games.get(gameId); exists {
		s.queueServerUpdatesAndSignal(clientId, s.createGameReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.Internal),
			ErrorMessage: "can not generate a game",
		}))
		return nil
	}

	game := &models.Game{
		Id:      gameId,
		Board:   [9]string{},
		Creator: creator,
		Result:  models.GameResult_INITIAL,
	}

	s.setupMover(game, player1, player2)

	s.playerGame.set(player1.Id, game.Id)
	s.playerGame.set(player2.Id, game.Id)
	s.games.set(game.Id, game)

	s.queueServerUpdatesAndSignal(clientId, s.createGameReply(&Outcome{Ok: true}))
	s.queueServerUpdatesAndSignal(player1ClientId,
		s.createNavigationUpdate(NavigationPath_GAME),
		s.createGameStartUpdate(game, player1, player2),
		s.createNextMoverUpdate(game),
	)
	s.queueServerUpdatesAndSignal(player2ClientId,
		s.createNavigationUpdate(NavigationPath_GAME),
		s.createGameStartUpdate(game, player2, player1),
		s.createNextMoverUpdate(game),
	)

	return nil
}
