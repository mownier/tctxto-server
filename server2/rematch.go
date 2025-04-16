package server2

import (
	"txtcto/models"

	"google.golang.org/grpc/codes"
)

func (s *Server) rematch(playerYouClientId string, in *RematchRequest) error {
	playerYou, outcome := s.validatePlayer(playerYouClientId)
	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(outcome))
		return nil
	}

	gameId, exists := s.playerGame.get(playerYou.Id)
	if !exists {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "you are not in a game",
		}))
		return nil
	}

	game, exists := s.games.get(gameId)
	if !exists {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "your game not found",
		}))
		return nil
	}

	if game.MoverO.Id != playerYou.Id || game.MoverX.Id != playerYou.Id {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.InvalidArgument),
			ErrorMessage: "you are not a game participant",
		}))
		return nil
	}

	//var playerOtherClientId string
	var playerOther *models.Player
	if game.MoverO.Id == playerYou.Id {
		_, playerOther, outcome = s.getClientIdAndPlayer(game.MoverX.Id, "other player")
	} else {
		_, playerOther, outcome = s.getClientIdAndPlayer(game.MoverO.Id, "other player")
	}

	if !outcome.Ok {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "other player details not found",
		}))
		return nil
	}

	if !in.Rematch {
		s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{Ok: true}))
		// TODO:
		return nil
	}

	gameRematch, exists := s.gameRematches.get(game.Id)
	if !exists {
		gameRematch = &models.GameRematch{}
		s.gameRematches.set(game.Id, gameRematch)
	}

	gameRematch.Players[playerYou.Id] = in.Rematch

	if _, exists := gameRematch.Players[playerOther.Id]; !exists {
		s.queueServerUpdatesAndSignal(playerYouClientId,
			s.createRematchReply(&Outcome{Ok: true}),
			// TODO
		)
		return nil
	}

	s.queueServerUpdatesAndSignal(playerYouClientId, s.createRematchReply(&Outcome{Ok: true}))

	s.playerGame.delete(playerYou.Id)
	s.playerGame.delete(playerOther.Id)

	if gameRematch.Players[playerYou.Id] && gameRematch.Players[playerOther.Id] {
		return s.createGame(playerYouClientId, &CreateGameRequest{Player1Id: playerYou.Id, Player2Id: playerOther.Id})
	}

	return nil
}
