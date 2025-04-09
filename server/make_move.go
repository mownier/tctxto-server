package server

import (
	context "context"
	"txtcto/models"
)

func (s *Server) MakeMove(ctx context.Context, in *MakeMoveRequest) (*Empty, error) {
	return &Empty{}, nil
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
	if game.Mover.Id == game.Player1.Id {
		game.Mover = game.Player2

	} else if game.Mover.Id == game.Player2.Id {
		game.Mover = game.Player1
	}
}
