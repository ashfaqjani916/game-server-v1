package game

// import (
// 	"encoding/json"
// 	"game-server-v1/pkg/types"
// 	"log"
// 	"time"
// )

// // const (
// // 	tickRate     = 20 // Server updates per second
// // 	tickInterval = time.Second / tickRate
// // 	moveSpeed    = 5.0  // Units per second
// // 	maxX         = 50.0 // World boundaries
// // 	maxY         = 50.0
// // 	minX         = -50.0
// // 	minY         = -50.0
// // )

// // func (h *GameHub) updateGameState() {
// //     now := time.Now()

// //     // Process all player inputs
// //     h.playersMux.Lock()
// //     for _, player := range h.players {
// //         h.applyPlayerInput(player, now)
// //     }
// //     h.playersMux.Unlock()

// //     // Process all projectiles
// //     h.projectilesMux.Lock()
// //     for id, proj := range h.projectiles {
// //         if proj.Expired(now) {
// //             delete(h.projectiles, id)
// //             continue
// //         }
// //         proj.Update(h.config.TickInterval)
// //         h.checkProjectileCollision(proj)
// //     }
// //     h.projectilesMux.Unlock()

// //     // Broadcast full snapshot (or diffs)
// //     h.broadcastWorldState()
// // }

// func (h *GameHub) GameLoop() {
// 	ticker := time.NewTicker(h.config.TickInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		// Tick-based updates
// 		case <-ticker.C:
// 			// this case is running based on ticker , here we have to call a function that updates the game state.
// 			now := time.Now()

// 			// Clean up inactive players
// 			h.playersMux.Lock()
// 			for playerID, player := range h.players {
// 				if now.Sub(player.LastUpdate) > 30*time.Second {
// 					log.Printf("Removing inactive player: %s", playerID)
// 					delete(h.players, playerID)
// 				}
// 			}
// 			h.playersMux.Unlock()

// 			// TODO: add physics, AI, projectiles, collisions here

// 		// New client registers
// 		case client := <-h.register:
// 			h.clientsMux.Lock()
// 			h.clients[client] = true
// 			h.clientsMux.Unlock()

// 			// Create player for this client
// 			player := &types.Player{
// 				ID:         client.UUID,
// 				PosX:       0,
// 				PosY:       0,
// 				MoveX:      0,
// 				MoveY:      0,
// 				FacingLeft: false,
// 				MoveSpeed:  h.config.MoveSpeed,
// 				LastUpdate: time.Now(),
// 			}

// 			client.Player = player

// 			h.playersMux.Lock()
// 			h.players[player.ID] = player
// 			h.playersMux.Unlock()

// 			// Send player ID back to client
// 			idMsg := types.PlayerIDMessage{
// 				Type:     "playerId",
// 				PlayerID: player.ID,
// 			}

// 			data, err := json.Marshal(idMsg)
// 			if err == nil {
// 				select {
// 				case client.Send <- data:
// 				default:
// 					close(client.Send)
// 					h.clientsMux.Lock()
// 					delete(h.clients, client)
// 					h.clientsMux.Unlock()
// 				}
// 			}

// 			log.Printf("Player %s connected", player.ID)

// 		// Client unregisters (disconnect)
// 		case client := <-h.unregister:
// 			h.clientsMux.Lock()
// 			if _, ok := h.clients[client]; ok {
// 				delete(h.clients, client)
// 				close(client.Send)
// 			}
// 			h.clientsMux.Unlock()

// 			if client.Player != nil {
// 				h.playersMux.Lock()
// 				delete(h.players, client.Player.ID)
// 				h.playersMux.Unlock()
// 				log.Printf("Player %s disconnected", client.Player.ID)
// 			}
// 		}
// 	}
// }
