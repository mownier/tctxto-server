package server

import (
	"context"
	"math/rand"
	"time"
	"txtcto/models"

	"google.golang.org/grpc/codes"
)

func (s *Server) MakeMove(ctx context.Context, in *MakeMoveRequest) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "make move cancelled")
	if err != nil {
		return nil, err
	}
	return s.makeMoveInternal(clientId, in.Position)
}

func (s *Server) makeMoveInternal(clientId string, position int32) (*Empty, error) {
	player, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		return &Empty{}, nil
	}

	s.lobbyGameMu.Lock()
	defer s.lobbyGameMu.Unlock()

	gameId, exists := s.playerGameMap[player.Id]
	if !exists {
		outcome = &Outcome{Ok: false, ErrorCode: int32(codes.NotFound), ErrorMessage: "player is not in a game"}
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		return &Empty{}, nil
	}

	game, exists := s.games[gameId]
	if !exists {
		outcome = &Outcome{Ok: false, ErrorCode: int32(codes.NotFound), ErrorMessage: "player's game details not found"}
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		return &Empty{}, nil
	}

	if position < 0 || int(position) >= len(game.Board) {
		outcome = &Outcome{Ok: false, ErrorCode: int32(codes.InvalidArgument), ErrorMessage: "position out of range"}
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		return &Empty{}, nil
	}

	if game.Board[position] != "" {
		outcome = &Outcome{Ok: false, ErrorCode: int32(codes.InvalidArgument), ErrorMessage: "position already occupied"}
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		return &Empty{}, nil
	}

	moverXClient, outcomeX := s.getPlayerAndValidateByPlayerID(game.MoverX.Id, "mover X")
	if !outcomeX.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcomeX)})
		return &Empty{}, nil
	}

	moverOClient, outcomeO := s.getPlayerAndValidateByPlayerID(game.MoverO.Id, "mover O")
	if !outcomeO.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcomeO)})
		return &Empty{}, nil
	}

	outcome.Ok = true
	game.Board[position] = player.Id

	if s.checkDraw(game) {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		s.queueUpdatesAndSignal(moverOClient.Id, []*SubscriptionUpdate{s.createDrawupdate()})
		s.queueUpdatesAndSignal(moverXClient.Id, []*SubscriptionUpdate{s.createDrawupdate()})
		return &Empty{}, nil
	}

	if s.checkWin(game) {
		winner, mover := s.determineWinner(game, player)
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
		s.queueUpdatesAndSignal(moverOClient.Id, []*SubscriptionUpdate{s.createWinnerUpdate(winner, mover)})
		s.queueUpdatesAndSignal(moverXClient.Id, []*SubscriptionUpdate{s.createWinnerUpdate(winner, mover)})
		return &Empty{}, nil
	}

	s.switchMover(game)
	s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createMakeMoveReply(outcome)})
	s.queueUpdatesAndSignal(moverOClient.Id, []*SubscriptionUpdate{s.createNextMoverUpdate(game)})
	s.queueUpdatesAndSignal(moverXClient.Id, []*SubscriptionUpdate{s.createNextMoverUpdate(game)})

	return &Empty{}, nil
}

func (s *Server) determineWinner(game *models.Game, lastPlayer *models.Player) (Winner, Mover) {
	var winner Winner
	var mover Mover

	if game.Mover.Id == lastPlayer.Id {
		winner = Winner_you
	} else {
		winner = Winner_other
	}

	if game.Mover.Id == game.MoverX.Id {
		mover = Mover_X
	} else if game.Mover.Id == game.MoverO.Id {
		mover = Mover_O
	} else {
		mover = Mover_UNSPECIFIED
	}
	return winner, mover
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
	} else if game.Mover.Id == game.MoverO.Id {
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
	} else {
		game.MoverO = player1
		game.MoverX = player2
		game.Mover = game.MoverO
	}
}
