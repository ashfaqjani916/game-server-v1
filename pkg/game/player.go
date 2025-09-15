package game

import (
	// "encoding/json"
	"game-server-v1/pkg/types"
	"log"
	"time"
)

// HandlePlayerMovement processes input from a client and updates the central GameState
func UpdatePlayerMovement(hub *GameHub, client *types.Client, input *types.PlayerInputMessage) {
	hub.state.mu.Lock()
	defer hub.state.mu.Unlock()

	// Ensure client has an associated player
	if client.Player == nil {
		log.Printf("Client %s has no associated player", client.UUID)
		return
	}

	playerID := client.Player.ID
	player, ok := hub.state.Players[playerID]
	if !ok {
		log.Printf("Player not found in GameState: %s", playerID)
		return
	}

	// Validate input (anti-cheat sanity check)
	if input.MoveX < -1 || input.MoveX > 1 || input.MoveY < -1 || input.MoveY > 1 {
		log.Printf("Invalid movement input from %s", playerID)
		return
	}

	// Calculate new position
	deltaTime := hub.config.TickInterval.Seconds()
	newX := player.PosX + input.MoveX*hub.config.MoveSpeed*deltaTime
	newY := player.PosY + input.MoveY*hub.config.MoveSpeed*deltaTime

	// Clamp position inside world bounds
	newX, newY = hub.config.WorldBounds.ClampPosition(newX, newY)

	// Update authoritative player state in GameState
	player.PosX = newX
	player.PosY = newY
	player.MoveX = input.MoveX
	player.MoveY = input.MoveY
	player.FacingLeft = input.FacingLeft
	player.LastUpdate = time.Now()

	// Update global GameState timestamp
	hub.state.LastUpdate = time.Now()
}

// // broadcastPlayerState sends updated player state to all connected clients
// func (h *GameHub) broadcastPlayerState(player *types.Player) {
// 	stateMsg := types.PlayerStateMessage{
// 		Type:       "playerState",
// 		PlayerID:   player.ID,
// 		PosX:       player.PosX,
// 		PosY:       player.PosY,
// 		MoveX:      player.MoveX,
// 		MoveY:      player.MoveY,
// 		FacingLeft: player.FacingLeft,
// 		Timestamp:  float64(time.Now().UnixNano()) / 1e9,
// 	}

// 	data, err := json.Marshal(stateMsg)
// 	if err != nil {
// 		log.Printf("Error marshaling player state: %v", err)
// 		return
// 	}

// 	// Push into hub’s broadcast channel
// 	select {
// 	case h.GetBroadcastChan() <- data:
// 	default:
// 		log.Printf("Broadcast channel full, dropping state update for %s", player.ID)
// 	}
// }
