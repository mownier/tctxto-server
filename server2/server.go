package server2

import (
	"context"
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
	clientLastNavigationPath    *safeMap[string, NavigationPath]
	clientPlayer                *safeMap[string, string]
	playerGame                  *safeMap[string, string]
	games                       *safeMap[string, *models.Game]
	playerNameId                *safeMap[string, string]
	playerClient                *safeMap[string, string]
	playerLobby                 *safeMap[string, string]
	lobbies                     *safeMap[string, *models.Lobby]

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
		clientLastNavigationPath:    newSafeMap[string, NavigationPath](),
		clientPlayer:                newSafeMap[string, string](),
		playerGame:                  newSafeMap[string, string](),
		games:                       newSafeMap[string, *models.Game](),
		playerNameId:                newSafeMap[string, string](),
		playerClient:                newSafeMap[string, string](),
		playerLobby:                 newSafeMap[string, string](),
		lobbies:                     newSafeMap[string, *models.Lobby](),
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
