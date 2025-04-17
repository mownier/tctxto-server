package server2

import (
	"txtcto/models"

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
		s.queueServerUpdatesAndSignal(playerYouClientId,
			s.createMakeMoveReply(&Outcome{Ok: true}),
			s.createWinnerUpdate(s.areYouTheMover(game, playerYou), Technicality_BY_FORFEIT),
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

		_, outcome = s.setupRematch(playerYou, playerOther)

		if outcome.Ok {
			s.queueServerUpdatesAndSignal(playerYouClientId,
				s.createMakeMoveReply(&Outcome{Ok: true}),
				s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
				s.createWinnerUpdate(s.areYouTheMover(game, playerYou), Technicality_NO_PROBLEM),
			)

			s.queueServerUpdatesAndSignal(playerOtherClientId,
				s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
				s.createWinnerUpdate(s.areYouTheMover(game, playerOther), Technicality_NO_PROBLEM),
			)

			return nil
		}

		s.queueServerUpdatesAndSignal(playerYouClientId, s.initialServerUpdates(playerYouClientId)...)
		s.queueServerUpdatesAndSignal(playerOtherClientId, s.initialServerUpdates(playerOtherClientId)...)

		return nil
	}

	if s.checkDraw(game) {
		game.Result = models.GameResult_DRAW

		_, outcome = s.setupRematch(playerYou, playerOther)

		if outcome.Ok {
			s.queueServerUpdatesAndSignal(playerYouClientId,
				s.createMakeMoveReply(&Outcome{Ok: true}),
				s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
				s.createDrawUpdate(),
			)

			s.queueServerUpdatesAndSignal(playerOtherClientId,
				s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
				s.createDrawUpdate(),
			)

			return nil
		}

		s.queueServerUpdatesAndSignal(playerYouClientId, s.initialServerUpdates(playerYouClientId)...)
		s.queueServerUpdatesAndSignal(playerOtherClientId, s.initialServerUpdates(playerOtherClientId)...)

		return nil
	}

	s.switchMover(game)

	s.queueServerUpdatesAndSignal(playerYouClientId,
		s.createMakeMoveReply(&Outcome{Ok: true}),
		s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
		s.createNextMoverUpdate(s.areYouTheMover(game, playerYou)),
	)

	s.queueServerUpdatesAndSignal(playerOtherClientId,
		s.createMoveUpdate(game, playerYou.Id, int32(in.Position)),
		s.createNextMoverUpdate(s.areYouTheMover(game, playerOther)),
	)

	return nil
}
