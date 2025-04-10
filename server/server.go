package server

import (
	context "context"
	"fmt"
	"sync"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type Server struct {
	publicKeys map[string]bool

	clients               map[string]*models.Client
	clientUpdatesMap      map[string][]*SubscriptionUpdate
	clientSignalMap       map[string]chan struct{}
	clientLastIndexUpdate map[string]int
	clientPlayerMap       map[string]string

	players         map[string]*models.Player
	playerClientMap map[string]string
	playerNameIdMap map[string]string
	playerLobbyMap  map[string]string
	playerGameMap   map[string]string

	lobbies map[string]*models.Lobby

	games map[string]*models.Game

	mu sync.RWMutex
	UnimplementedTicTacToeServer
}

func NewServer() *Server {
	publicKeys := make(map[string]bool)
	publicKeys["12345"] = true
	return &Server{
		publicKeys: publicKeys,

		clients:               make(map[string]*models.Client),
		clientUpdatesMap:      make(map[string][]*SubscriptionUpdate),
		clientSignalMap:       make(map[string]chan struct{}),
		clientLastIndexUpdate: make(map[string]int),
		clientPlayerMap:       make(map[string]string),

		players:         make(map[string]*models.Player),
		playerClientMap: make(map[string]string),
		playerNameIdMap: make(map[string]string),
		playerLobbyMap:  make(map[string]string),
		playerGameMap:   make(map[string]string),

		lobbies: make(map[string]*models.Lobby),

		games: make(map[string]*models.Game),
	}
}

func (s *Server) extractClientId(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.NotFound, "metadata not ok")
	}
	values := md.Get("ClientId")
	if len(values) == 0 {
		return "", status.Error(codes.NotFound, "client not found")
	}
	clientId := values[0]
	if clientId == "" {
		return "", status.Error(codes.InvalidArgument, "client is empty")
	}
	return clientId, nil
}

func (s *Server) extractClientIdWithCancel(ctx context.Context, cancelMessage string) (string, error) {
	select {
	case <-ctx.Done():
		return "", status.Error(codes.Canceled, cancelMessage)
	default:
		return s.extractClientId(ctx)
	}
}

func (s *Server) getPlayerAndValidate(clientID string) (*models.Player, *Outcome) {
	if _, exists := s.clients[clientID]; !exists {
		return nil, &Outcome{ErrorCode: int32(codes.NotFound), ErrorMessage: "unknown client"}
	}
	return s.checkPlayer(clientID)
}

func (s *Server) getPlayerAndValidateByPlayerID(playerID string, playerName string) (*models.Player, *Outcome) {
	clientID, exists := s.playerClientMap[playerID]
	if !exists {
		return nil, &Outcome{ErrorCode: int32(codes.Internal), ErrorMessage: fmt.Sprintf("client ID for %s not found", playerName)}
	}
	player, outcome := s.checkPlayer(clientID)
	if !outcome.Ok {
		outcome.ErrorMessage = fmt.Sprintf("%s error: %s", playerName, outcome.ErrorMessage)
	}
	return player, outcome
}

func (s *Server) checkPlayer(clientId string) (*models.Player, *Outcome) {
	s.mu.Lock()
	defer s.mu.Unlock()

	outcome := &Outcome{}

	playerId, exists := s.clientPlayerMap[clientId]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.Unauthenticated)
		outcome.ErrorMessage = "client has no authenticated player"

		return &models.Player{Id: ""}, outcome
	}

	if _, exists := s.playerClientMap[playerId]; !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.Unauthenticated)
		outcome.ErrorMessage = "player is not authenticated in the client"

		return &models.Player{Id: playerId}, outcome
	}

	player, exists := s.players[playerId]

	if !exists {
		outcome.Ok = false
		outcome.ErrorCode = int32(codes.NotFound)
		outcome.ErrorMessage = "player not found"

		return &models.Player{Id: playerId}, outcome
	}

	outcome.Ok = true

	return player, outcome
}
