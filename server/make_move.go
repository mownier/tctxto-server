package server

import (
	context "context"
	"math/rand"
	"time"
	"txtcto/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) MakeMove(ctx context.Context, in *MakeMoveRequest) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "make move cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.makeMove(clientId, in.Position)
	}
}

func (s *Server) makeMove(clientId string, position int32) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	player, outcome := s.checkPlayer(clientId)

	if !outcome.Ok {
		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	gameId, exists := s.playerGameMap[player.Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "player is not in a game"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	game, exists := s.games[gameId]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "player's game details not found"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	if position < 0 || int(position) >= len(game.Board) {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.InvalidArgument)
		outcome.ErrorMessage = "position out of range"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	if game.Board[position] != "" || len(game.Board[position]) > 0 {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.InvalidArgument)
		outcome.ErrorMessage = "position already occupied"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	moverXClientId, exists := s.playerClientMap[game.MoverX.Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "client not found for mover X"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	moverOClientId, exists := s.playerClientMap[game.MoverO.Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "client not found for mover O"

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)

		return &Empty{}, nil
	}

	outcome = &Outcome{Ok: true}

	game.Board[position] = player.Id

	if s.checkDraw(game) {
		s.queueSignalUpdatesOnMakeMove(clientId, outcome)
		s.queueSignalUpdatesOnDraw(moverOClientId, game)
		s.queueSignalUpdatesOnDraw(moverXClientId, game)

		return &Empty{}, nil
	}

	if s.checkWin(game) {
		var playerYou *models.Player
		var playerOther *models.Player

		if game.MoverO.Id == player.Id {
			playerYou = game.MoverO
			playerOther = game.MoverX
		}

		if game.MoverX.Id == player.Id {
			playerYou = game.MoverX
			playerOther = game.MoverO
		}

		var winner Winner
		var mover Mover

		if game.Mover.Id == playerYou.Id {
			winner = Winner_you
		}

		if game.Mover.Id == playerOther.Id {
			winner = Winner_other
		}

		if game.Mover.Id == game.MoverX.Id {
			mover = Mover_X
		}

		if game.Mover.Id == game.MoverO.Id {
			mover = Mover_O
		}

		s.queueSignalUpdatesOnMakeMove(clientId, outcome)
		s.queueSignalUpdatesOnWinner(moverOClientId, winner, mover)
		s.queueSignalUpdatesOnWinner(moverXClientId, winner, mover)

		return &Empty{}, nil
	}

	s.switchMover(game)

	s.queueSignalUpdatesOnMakeMove(clientId, outcome)
	s.queueSignalUpdatesOnNextMover(moverOClientId, game)
	s.queueSignalUpdatesOnNextMover(moverXClientId, game)

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnMakeMove(clientId string, outcome *Outcome) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createMakeMoveReply(outcome),
	)

	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:
			break

		default:
			break
		}
	}
}

func (s *Server) queueSignalUpdatesOnWinner(clientId string, winner Winner, mover Mover) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createWinnerUpdate(winner, mover),
	)

	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:
			break

		default:
			break
		}
	}
}

func (s *Server) queueSignalUpdatesOnDraw(clientId string, game *models.Game) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createDrawupdate(),
	)

	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:
			break

		default:
			break
		}
	}
}

func (s *Server) checkDraw(game *models.Game) bool {
	for _, tile := range game.Board {
		if tile == "" {
			return false
		}
	}

	return true
}

func (s *Server) checkWin(game *models.Game) bool {
	wins := [][]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // Rows
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // Columns
		{0, 4, 8}, {2, 4, 6}, // Diagonals
	}

	for _, win := range wins {
		if game.Board[win[0]] != "" && game.Board[win[0]] == game.Board[win[1]] && game.Board[win[1]] == game.Board[win[2]] {
			return true
		}
	}

	return false
}

func (s *Server) switchMover(game *models.Game) {
	if game.Mover.Id == game.MoverX.Id {
		game.Mover = game.MoverO
		return
	}

	if game.Mover.Id == game.MoverO.Id {
		game.Mover = game.MoverX
	}
}

func (s *Server) setupMover(game *models.Game, player1 *models.Player, player2 *models.Player) {
	source := rand.NewSource(time.Now().UnixNano())

	r := rand.New(source)

	if r.Intn(2) == 1 {
		game.MoverX = player1
		game.MoverO = player2
		game.Mover = game.MoverX
		return
	}

	game.MoverO = player1
	game.MoverX = player2
	game.Mover = game.MoverO
}
