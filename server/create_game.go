package server

import context "context"

func (s *Server) CreateGame(context.Context, *CreateGameRequest) (*Empty, error) {
	return &Empty{}, nil
}
