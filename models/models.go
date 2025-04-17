package models

type Consumer struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name"`
}

type Client struct {
	Id string
}

type Lobby struct {
	Id      string
	Name    string
	Creator *Player
	Players map[string]*Player
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

type Rematch struct {
	Id              string
	PlayerDecisions [2]*PlayerDecision
}

type PlayerDecision struct {
	Player   *Player
	Decision Decision
}

type GameResult int32

const (
	GameResult_INITIAL        GameResult = 0
	GameResult_ONGOING        GameResult = 1
	GameResult_WIN            GameResult = 2
	GameResult_DRAW           GameResult = 3
	GameResult_WIN_BY_FORFEIT GameResult = 4
)

type Decision int32

const (
	Decision_UNDECIDED Decision = 0
	Decision_YES       Decision = 1
	Decision_NO        Decision = 2
)

func (r *Rematch) Confirmed() bool {
	for _, pd := range r.PlayerDecisions {
		if pd.Decision == Decision_UNDECIDED || pd.Decision == Decision_NO {
			return false
		}
	}
	return true
}

func (r *Rematch) Cancelled() bool {
	for _, pd := range r.PlayerDecisions {
		if pd.Decision == Decision_NO {
			return true
		}
	}
	return false
}

func (r *Rematch) Pending() bool {
	for _, pd := range r.PlayerDecisions {
		if pd.Decision == Decision_NO {
			return false
		}
	}
	return true
}

func (r *Rematch) GetPlayerDecision(playerId string) (*PlayerDecision, bool) {
	for _, pd := range r.PlayerDecisions {
		if pd.Player.Id == playerId {
			return pd, true
		}
	}
	return nil, false
}

func (r *Rematch) SetPlayerDecision(playerId string, decision Decision) {
	for _, pd := range r.PlayerDecisions {
		if pd.Player.Id == playerId {
			pd.Decision = decision
			break
		}
	}
}
