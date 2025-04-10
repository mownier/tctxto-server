package server

import (
	context "context"
	"fmt"
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (s *Server) CreateGame(ctx context.Context, in *CreateGameRequest) (*Empty, error) {
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "create game was cancelled")

	default:
		clientId, err := s.extractClientId(ctx)

		if err != nil {
			return nil, err
		}

		return s.createGame(clientId, in)
	}
}

func (s *Server) createGame(clientId string, in *CreateGameRequest) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientId]; !exists {
		return nil, status.Error(codes.NotFound, "unknown client")
	}

	creator, outcome := s.checkPlayer(clientId)

	if !outcome.Ok {
		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	if _, exists := s.playerGameMap[creator.Id]; exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "creator is currently in a game"

		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	player1ClientId, exists := s.playerClientMap[in.Player1Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.Internal)
		outcome.ErrorMessage = "client for player 1 not set"

		return &Empty{}, nil
	}

	player1, outcome := s.checkPlayer(player1ClientId)

	if !outcome.Ok {
		outcome.ErrorMessage = fmt.Sprintf("player 1 error: %s", outcome.ErrorMessage)

		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	if _, exists := s.playerGameMap[in.Player1Id]; exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "player 1 is currently in a game"

		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	player2ClientId, exists := s.playerClientMap[in.Player2Id]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.Internal)
		outcome.ErrorMessage = "client for player 2 not set"

		return &Empty{}, nil
	}

	player2, outcome := s.checkPlayer(player2ClientId)

	if !outcome.Ok {
		outcome.ErrorMessage = fmt.Sprintf("player 2 error: %s", outcome.ErrorMessage)

		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	if _, exists := s.playerGameMap[in.Player2Id]; exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.AlreadyExists)
		outcome.ErrorMessage = "player 2 is currently in a game"

		s.queueSignalUpdatesOnCreateGame(clientId, outcome)

		return &Empty{}, nil
	}

	const maxAttempt = 10

	for i := 0; i < maxAttempt; i++ {
		id := uuid.New().String()

		if _, exists := s.games[id]; !exists {
			game := &models.Game{
				Id:      id,
				Board:   [9]string{},
				Creator: creator,
				Result:  models.GameResult_INITIAL,
			}

			s.setupMover(game, player1, player2)

			s.playerGameMap[player1.Id] = id
			s.playerGameMap[player2.Id] = id

			outcome = &Outcome{Ok: true}

			s.queueSignalUpdatesOnCreateGame(clientId, outcome)
			s.queueSignalUpdatesOnGameStart(player1ClientId, game, player1, player2)
			s.queueSignalUpdatesOnGameStart(player2ClientId, game, player2, player1)
			s.queueSignalUpdatesOnNextMover(player1ClientId, game)
			s.queueSignalUpdatesOnNextMover(player2ClientId, game)

			return &Empty{}, nil
		}
	}

	return &Empty{}, nil
}

func (s *Server) queueSignalUpdatesOnCreateGame(clientId string, outcome *Outcome) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createCreateGameReply(outcome),
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

func (s *Server) queueSignalUpdatesOnGameStart(clientId string, game *models.Game, you, other *models.Player) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createNavigationUpdate(NavigationPath_GAME),
		s.createGameStartUpdate(game, you, other),
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

func (s *Server) queueSignalUpdatesOnNextMover(clientId string, game *models.Game) {
	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId],
		s.createNextMoverUpdate(game),
	)
}
