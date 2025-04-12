package server

import (
	context "context"
	"fmt"
	"log"
	"sync"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type Server struct {
	consumerMu sync.RWMutex
	consumers  map[string]*models.Consumer

	clientSubscriptionMu  sync.RWMutex
	clientSignalMap       map[string]chan struct{}
	clientUpdatesMap      map[string][]*SubscriptionUpdate
	clientLastIndexUpdate map[string]int

	playerDataMu    sync.RWMutex
	clients         map[string]*models.Client
	clientPlayerMap map[string]string
	playerClientMap map[string]string
	players         map[string]*models.Player
	playerNameIdMap map[string]string

	lobbyGameMu    sync.RWMutex
	lobbies        map[string]*models.Lobby
	playerLobbyMap map[string]string
	games          map[string]*models.Game
	playerGameMap  map[string]string

	activeSubscriptionsMu sync.RWMutex
	activeSubscriptions   map[string]bool

	clientLastNavPathUpdateMu sync.RWMutex
	clientLastNavPathUpdate   map[string]NavigationPath

	UnimplementedTicTacToeServer
}

func NewServer(consumers map[string]*models.Consumer) *Server {
	return &Server{
		consumers: consumers,

		clients:                 make(map[string]*models.Client),
		clientUpdatesMap:        make(map[string][]*SubscriptionUpdate),
		clientSignalMap:         make(map[string]chan struct{}),
		clientLastIndexUpdate:   make(map[string]int),
		clientPlayerMap:         make(map[string]string),
		clientLastNavPathUpdate: make(map[string]NavigationPath),

		players:         make(map[string]*models.Player),
		playerClientMap: make(map[string]string),
		playerNameIdMap: make(map[string]string),
		playerLobbyMap:  make(map[string]string),
		playerGameMap:   make(map[string]string),

		lobbies: make(map[string]*models.Lobby),

		games: make(map[string]*models.Game),

		activeSubscriptions: make(map[string]bool),
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
	s.playerDataMu.RLock()
	_, exists := s.clients[clientID]
	s.playerDataMu.RUnlock()

	if !exists {
		return nil, &Outcome{ErrorCode: int32(codes.NotFound), ErrorMessage: "unknown client"}
	}

	return s.checkPlayer(clientID)
}

func (s *Server) getPlayerAndValidateByPlayerID(playerID string, playerName string) (*models.Player, *Outcome) {
	s.playerDataMu.RLock()
	clientID, exists := s.playerClientMap[playerID]
	s.playerDataMu.RUnlock()

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
	s.playerDataMu.Lock()
	defer s.playerDataMu.Unlock()

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

func (s *Server) queueUpdatesAndSignal(clientId string, updates []*SubscriptionUpdate) {
	log.Printf("queueUpdatesAndSignal = %s, updates = %d\n", clientId, len(updates))

	s.clientSubscriptionMu.Lock()
	defer s.clientSubscriptionMu.Unlock()

	log.Printf("queueUpdatesAndSignal = %s, updates = %d, LOCK\n", clientId, len(updates))

	if _, exists := s.clientUpdatesMap[clientId]; !exists {
		s.clientUpdatesMap[clientId] = []*SubscriptionUpdate{}
	}

	s.clientUpdatesMap[clientId] = append(s.clientUpdatesMap[clientId], updates...)

	if signal, exists := s.clientSignalMap[clientId]; exists {
		select {
		case signal <- struct{}{}:
			// Signal sent
		default:
			// Non-blocking send if the channel is full
		}
	}

	log.Printf("queueUpdatesAndSignal = %s, updates = %d, UNLOCK\n", clientId, len(updates))
}
