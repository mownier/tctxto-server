package server2

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"txtcto/models"

	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type Server struct {
	consumers                   *safeMap[string, *models.Consumer]
	clients                     *safeMap[string, *models.Client]
	players                     *safeMap[string, *models.Player]
	clientSignal                *safeMap[string, chan struct{}]
	clientServerUpdates         *safeMap[string, []*ServerUpdate]
	clientLastIndexServerUpdate *safeMap[string, int]
	clientPlayer                *safeMap[string, string]
	playerGame                  *safeMap[string, string]
	games                       *safeMap[string, *models.Game]
	playerNameId                *safeMap[string, string]
	playerClient                *safeMap[string, string]
	playerLobby                 *safeMap[string, string]
	lobbies                     *safeMap[string, *models.Lobby]
	rematches                   *safeMap[string, *models.Rematch]
	playerRematch               *safeMap[string, string]
	playerSearchingLobby        *safeMap[string, bool]

	UnimplementedTicTacToeServer
}

func NewServer(consumers map[string]*models.Consumer) *Server {
	return &Server{
		consumers:                   newSafeMapWith(consumers),
		clients:                     newSafeMap[string, *models.Client](),
		players:                     newSafeMap[string, *models.Player](),
		clientSignal:                newSafeMap[string, chan struct{}](),
		clientServerUpdates:         newSafeMap[string, []*ServerUpdate](),
		clientLastIndexServerUpdate: newSafeMap[string, int](),
		clientPlayer:                newSafeMap[string, string](),
		playerGame:                  newSafeMap[string, string](),
		games:                       newSafeMap[string, *models.Game](),
		playerNameId:                newSafeMap[string, string](),
		playerClient:                newSafeMap[string, string](),
		playerLobby:                 newSafeMap[string, string](),
		lobbies:                     newSafeMap[string, *models.Lobby](),
		rematches:                   newSafeMap[string, *models.Rematch](),
		playerRematch:               newSafeMap[string, string](),
		playerSearchingLobby:        newSafeMap[string, bool](),
	}
}

func (s *Server) extractPublicKeyWithCancel(ctx context.Context, cancelMessage string) (string, error) {
	select {
	case <-ctx.Done():
		return "", status.Error(codes.Canceled, cancelMessage)
	default:
		return s.extractPublicKey(ctx)
	}
}

func (s *Server) extractPublicKey(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.NotFound, "metadata not ok")
	}
	values := md.Get("PublicKey")
	if len(values) == 0 {
		return "", status.Error(codes.NotFound, "public key not found")
	}
	clientId := values[0]
	if clientId == "" {
		return "", status.Error(codes.InvalidArgument, "public key is empty")
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

func (s *Server) queueServerUpdatesAndSignal(clientId string, updates ...*ServerUpdate) {
	list, exists := s.clientServerUpdates.get(clientId)
	if !exists {
		list = []*ServerUpdate{}
	}
	s.clientServerUpdates.set(clientId, append(list, updates...))

	if signal, exists := s.clientSignal.get(clientId); exists {
		select {
		case signal <- struct{}{}:
			// Signal sent
		default:
			// Non-blocking send if the channel is full
		}
	}
}

func (s *Server) generateRandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	return string(b)
}

func (s *Server) validatePlayer(clientId string) (*models.Player, *Outcome) {
	playerId, exists := s.clientPlayer.get(clientId)
	if !exists {
		return nil, &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player not found",
		}
	}

	player, exists := s.players.get(playerId)
	if !exists {
		return nil, &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: "player details not found",
		}
	}

	return player, &Outcome{Ok: true}
}

func (s *Server) getClientIdAndPlayer(playerId string, alias string) (string, *models.Player, *Outcome) {
	player, exists := s.players.get(playerId)
	if !exists {
		return "", nil, &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: fmt.Sprintf("%s details not found", alias),
		}
	}

	clientId, exists := s.playerClient.get(player.Id)
	if !exists {
		return "", nil, &Outcome{
			Ok:           false,
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: fmt.Sprintf("client for %s found", alias),
		}
	}

	return clientId, player, &Outcome{Ok: true}
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

func (s *Server) areYouTheMover(game *models.Game, you *models.Player) bool {
	return game.Mover.Id == you.Id
}
