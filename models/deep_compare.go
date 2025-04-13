package models

import "reflect"

func DeepCompareLobby(l1, l2 *Lobby) bool {
	if l1 == nil && l2 == nil {
		return true
	}
	if l1 == nil || l2 == nil {
		return false
	}

	if l1.Id != l2.Id || l1.Name != l2.Name {
		return false
	}

	// Compare Creator
	if !DeepComparePlayer(l1.Creator, l2.Creator) {
		return false
	}

	// Compare Players map
	if len(l1.Players) != len(l2.Players) {
		return false
	}
	for key, player1 := range l1.Players {
		player2, ok := l2.Players[key]
		if !ok || !DeepComparePlayer(player1, player2) {
			return false
		}
	}

	// Compare AssignedIds map
	if !reflect.DeepEqual(l1.AssignedIds, l2.AssignedIds) {
		return false
	}

	// Compare PlayerAssignedId map
	if !reflect.DeepEqual(l1.PlayerAssignedId, l2.PlayerAssignedId) {
		return false
	}

	return true
}

// DeepComparePlayer compares two *Player instances for deep equality.
func DeepComparePlayer(p1, p2 *Player) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	return p1.Id == p2.Id &&
		p1.Name == p2.Name &&
		p1.Pass == p2.Pass &&
		p1.DisplayName == p2.DisplayName
}
