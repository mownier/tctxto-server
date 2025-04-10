package server

import (
	context "context"
	"fmt"
	"txtcto/models"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

func (s *Server) CreateGame(ctx context.Context, in *CreateGameRequest) (*Empty, error) {
	clientId, err := s.extractClientIdWithCancel(ctx, "create game was cancelled")
	if err != nil {
		return nil, err
	}
	return s.createGameInternal(clientId, in)
}

func (s *Server) createGameInternal(clientId string, in *CreateGameRequest) (*Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	creator, outcome := s.getPlayerAndValidate(clientId)
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
		return &Empty{}, nil
	}

	if _, exists := s.playerGameMap[creator.Id]; exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "creator is currently in a game",
		})})
		return &Empty{}, nil
	}

	player1, outcome := s.getPlayerAndValidateByPlayerID(in.Player1Id, "player 1")
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
		return &Empty{}, nil
	}
	if _, exists := s.playerGameMap[in.Player1Id]; exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player 1 is currently in a game",
		})})
		return &Empty{}, nil
	}

	player2, outcome := s.getPlayerAndValidateByPlayerID(in.Player2Id, "player 2")
	if !outcome.Ok {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
		return &Empty{}, nil
	}
	if _, exists := s.playerGameMap[in.Player2Id]; exists {
		s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(&Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.AlreadyExists),
			ErrorMessage: "player 2 is currently in a game",
		})})
		return &Empty{}, nil
	}

	const maxAttempt = 10
	for i := 0; i < maxAttempt; i++ {
		gameID := uuid.New().String()
		if _, exists := s.games[gameID]; !exists {
			game := &models.Game{
				Id:      gameID,
				Board:   [9]string{},
				Creator: creator,
				Result:  models.GameResult_INITIAL,
			}
			s.setupMover(game, player1, player2)
			s.playerGameMap[player1.Id] = gameID
			s.playerGameMap[player2.Id] = gameID

			outcome := &Outcome{Ok: true}

			player1ClientId, ok1 := s.playerClientMap[in.Player1Id]
			if !ok1 {
				outcome.Ok = false
				outcome.ErrorCode = int32(codes.Internal)
				outcome.ErrorMessage = fmt.Sprintf("internal error: client ID not found for player 1: %s", in.Player1Id)
				s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
				return &Empty{}, nil
			}
			player2ClientId, ok2 := s.playerClientMap[in.Player2Id]
			if !ok2 {
				outcome.Ok = false
				outcome.ErrorCode = int32(codes.Internal)
				outcome.ErrorMessage = fmt.Sprintf("internal error: client ID not found for player 2: %s", in.Player2Id)
				s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
				return &Empty{}, nil
			}

			s.queueUpdatesAndSignal(clientId, []*SubscriptionUpdate{s.createCreateGameReply(outcome)})
			s.queueUpdatesAndSignal(player1ClientId, []*SubscriptionUpdate{
				s.createNavigationUpdate(NavigationPath_GAME),
				s.createGameStartUpdate(game, player1, player2),
				s.createNextMoverUpdate(game),
			})
			s.queueUpdatesAndSignal(player2ClientId, []*SubscriptionUpdate{
				s.createNavigationUpdate(NavigationPath_GAME),
				s.createGameStartUpdate(game, player2, player1),
				s.createNextMoverUpdate(game),
			})

			return &Empty{}, nil
		}
	}

	return &Empty{}, nil
}
