package models

func (l *Lobby) DeepCopy() *Lobby {
	if l == nil {
		return nil
	}

	newLobby := &Lobby{
		Id:               l.Id,
		Name:             l.Name,
		AssignedIds:      make(map[string]string),
		PlayerAssignedId: make(map[string]string),
		Players:          make(map[string]*Player),
	}

	// Deep copy Creator
	if l.Creator != nil {
		newLobby.Creator = &Player{
			Id:          l.Creator.Id,
			Name:        l.Creator.Name,
			Pass:        l.Creator.Pass,
			DisplayName: l.Creator.DisplayName,
		}
	}

	// Deep copy Players map
	for key, player := range l.Players {
		if player != nil {
			newLobby.Players[key] = &Player{
				Id:          player.Id,
				Name:        player.Name,
				Pass:        player.Pass,
				DisplayName: player.DisplayName,
			}
		}
	}

	// Copy AssignedIds map (strings are value types)
	for key, value := range l.AssignedIds {
		newLobby.AssignedIds[key] = value
	}

	// Copy PlayerAssignedId map (strings are value types)
	for key, value := range l.PlayerAssignedId {
		newLobby.PlayerAssignedId[key] = value
	}

	return newLobby
}
