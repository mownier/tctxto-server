package models

type Consumer struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name"`
}

type Client struct {
	Id string
}

type Lobby struct {
	Id               string
	Name             string
	Creator          *Player
	Players          map[string]*Player
	AssignedIds      map[string]string
	PlayerAssignedId map[string]string
}

type Player struct {
	Id          string
	Name        string
	Pass        string
	DisplayName string
}

type Game struct {
	Id      string
	Board   [9]string
	Creator *Player
	Mover   *Player
	MoverX  *Player
	MoverO  *Player
	Winner  *Player
	Result  GameResult
}

type GameResult int32

const (
	GameResult_INITIAL GameResult = 0
	GameResult_ONGOING GameResult = 1
	GameResult_WIN     GameResult = 2
	GameResult_DRAW    GameResult = 3
)
