package server2

import (
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func (s *Server) makeMove(playerYouClientId string, in *MakeMoveRequest) error {
	playerYou, outcome := s.validatePlayer(playerYouClientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(outcome))
		return nil
	}

	gameId, exists := s.playerGame.get(playerYou.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "you are not in a game",
		}))
		return nil
	}

	game, exists := s.games.get(gameId)
	if !exists {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "your game not found",
		}))
		return nil
	}

	if game.Result == models.GameResult_DRAW || game.Result == models.GameResult_WIN || game.Result == models.GameResult_WIN_BY_FORFEIT {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "your game has already ended",
		}))
		return nil
	}

	if game.MoverO.Id != playerYou.Id && game.MoverX.Id != playerYou.Id {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "you are not a game participant",
		}))
		return nil
	}

	if game.Mover.Id != playerYou.Id {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "it is not your turn to move",
		}))
		return nil
	}

	var playerOther *models.Player
	var playerOtherClientId string
	if game.MoverO.Id == playerYou.Id {
		playerOtherClientId, playerOther, outcome = s.getClientIdAndPlayer(game.MoverX.Id, "other player")
	} else {
		playerOtherClientId, playerOther, outcome = s.getClientIdAndPlayer(game.MoverO.Id, "other player")
	}
	if !outcome.Ok {
		game.Result = models.GameResult_WIN_BY_FORFEIT
		winner, mover := s.determineWinner(game, playerYou)
		s.queueServerUpdatesAndSignal(playerYouClientId,
			s.createMakeMoveReply(&Outcome{Ok: true}),
			s.createWinnerUpdate(mover, winner, Technicality_BY_FORFEIT),
		)
		return nil
	}

	if in.Position < 0 || int(in.Position) >= len(game.Board) {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "your move postiion is out of range",
		}))
		return nil
	}

	if game.Board[in.Position] != "" {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createMakeMoveReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "your move is not valid because the position is already occupied",
		}))
		return nil
	}

	game.Result = models.GameResult_ONGOING
	game.Board[in.Position] = playerYou.Id

	if s.checkWin(game) {
		game.Result = models.GameResult_WIN
		winner, mover := s.determineWinner(game, playerYou)
		s.queueServerUpdatesAndSignal(playerYouClientId,
			s.createMakeMoveReply(&Outcome{Ok: true}),
			s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
			s.createWinnerUpdate(mover, winner, Technicality_NO_PROBLEM),
		)
		winner, mover = s.determineWinner(game, playerOther)
		s.queueServerUpdatesAndSignal(playerOtherClientId,
			s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
			s.createWinnerUpdate(mover, winner, Technicality_NO_PROBLEM),
		)

		rematchId := uuid.New().String()

		if _, exists := s.rematches.get(rematchId); exists {
			updates := []*ServerUpdate{}
			updates = append(updates, s.initialServerUpdates(playerOtherClientId)...)
			s.queueServerUpdatesAndSignal(playerOtherClientId, updates...)

			updates = []*ServerUpdate{}
			updates = append(updates, s.initialServerUpdates(playerYouClientId)...)
			s.queueServerUpdatesAndSignal(playerYouClientId, updates...)

			return nil
		}

		rematch := &models.Rematch{
			Id:              rematchId,
			PlayerDecisions: [2]*models.PlayerDecision{},
		}
		rematch.PlayerDecisions[0] = &models.PlayerDecision{
			Player:   playerYou,
			Decision: models.Decision_UNDECIDED,
		}
		rematch.PlayerDecisions[1] = &models.PlayerDecision{
			Player:   playerOther,
			Decision: models.Decision_UNDECIDED,
		}
		s.playerRematch.set(playerYou.Id, rematch.Id)
		s.playerRematch.set(playerOther.Id, rematch.Id)
		s.rematches.set(rematch.Id, rematch)

		return nil
	}

	if s.checkDraw(game) {
		game.Result = models.GameResult_DRAW
		s.queueServerUpdatesAndSignal(playerYouClientId,
			s.createMakeMoveReply(&Outcome{Ok: true}),
			s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
			s.createDrawUpdate(),
		)
		s.queueServerUpdatesAndSignal(playerOtherClientId,
			s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
			s.createDrawUpdate(),
		)

		rematchId := uuid.New().String()

		if _, exists := s.rematches.get(rematchId); exists {
			updates := []*ServerUpdate{}
			updates = append(updates, s.initialServerUpdates(playerOtherClientId)...)
			s.queueServerUpdatesAndSignal(playerOtherClientId, updates...)

			updates = []*ServerUpdate{}
			updates = append(updates, s.initialServerUpdates(playerYouClientId)...)

			s.queueServerUpdatesAndSignal(playerYouClientId, updates...)
			return nil
		}

		rematch := &models.Rematch{
			Id:              rematchId,
			PlayerDecisions: [2]*models.PlayerDecision{},
		}
		rematch.PlayerDecisions[0] = &models.PlayerDecision{
			Player:   playerYou,
			Decision: models.Decision_UNDECIDED,
		}
		rematch.PlayerDecisions[1] = &models.PlayerDecision{
			Player:   playerOther,
			Decision: models.Decision_UNDECIDED,
		}
		s.playerRematch.set(playerYou.Id, rematch.Id)
		s.playerRematch.set(playerOther.Id, rematch.Id)
		s.rematches.set(rematch.Id, rematch)

		return nil
	}

	s.switchMover(game)

	s.queueServerUpdatesAndSignal(playerYouClientId,
		s.createMakeMoveReply(&Outcome{Ok: true}),
		s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
		s.createNextMoverUpdate(game),
	)
	s.queueServerUpdatesAndSignal(playerOtherClientId,
		s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
		s.createNextMoverUpdate(game),
	)

	return nil
}
