package network

import (
	"encoding/json"
	"fmt"
	"game-server-v1/pkg/game"
	"game-server-v1/pkg/types"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ReadPump(hub *game.GameHub, c *types.Client) {
	defer func() {
		// unregister the client when the loop exits
		hub.GetUnregisterChan() <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected close from %s: %v", c.UUID, err)
			} else {
				log.Printf("read error from %s: %v", c.UUID, err)
			}
			break
		}

		// update last-seen timestamp
		c.LastSeen = time.Now()

		// Try to parse message into a known type
		var base types.BaseMessage
		if err := json.Unmarshal(message, &base); err != nil {
			log.Printf("invalid message from %s: %v", c.UUID, err)
			continue
		}

		switch base.Type {
		case "playerInput":
			var input types.PlayerInputMessage
			if err := json.Unmarshal(message, &input); err != nil {
				log.Printf("invalid player input from %s: %v", c.UUID, err)
				continue
			}
			// Push into hub’s channel (central store will handle it in the game loop)
			hub.GetPlayerInputChan() <- &input

		case "shoot":
			var projMsg types.ProjectileMessage
			if err := json.Unmarshal(message, &projMsg); err != nil {
				log.Printf("invalid projectile message from %s: %v", c.UUID, err)
				continue
			}
			// Convert to Projectile and push into hub (you’d implement in hub/game state)
			hub.AddProjectileFromClient(c, &projMsg)

		default:
			log.Printf("unrecognized message type %s from %s", base.Type, c.UUID)
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
//
// A goroutine running WritePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func WritePump(client *types.Client) {
	ticker := time.NewTicker(types.PingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))
			if !ok {
				// The hub closed the channel.
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting writer for client %s: %v", client.UUID, err)
				return
			}

			// Write the message
			w.Write(message)

			// Add queued messages to the current websocket message if any
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				additionalMessage := <-client.Send
				w.Write(additionalMessage)
			}

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer for client %s: %v", client.UUID, err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed for client %s: %v", client.UUID, err)
				return
			}
		}
	}
}

// Alternative WritePump that handles GameState messages specifically
func WritePumpWithGameState(client *types.Client) {
	ticker := time.NewTicker(types.PingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
		log.Printf("WritePump closed for client %s", client.UUID)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// The hub closed the channel.
				client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))

			// Send the message (this could be a GameState update or any other message)
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing message to client %s: %v", client.UUID, err)
				return
			}

			// Process any additional queued messages
			// This is important for GameState updates to prevent message buildup
			n := len(client.Send)
			for i := 0; i < n; i++ {
				select {
				case additionalMessage := <-client.Send:
					if err := client.Conn.WriteMessage(websocket.TextMessage, additionalMessage); err != nil {
						log.Printf("Error writing additional message to client %s: %v", client.UUID, err)
						return
					}
				default:
					break
				}
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed for client %s: %v", client.UUID, err)
				return
			}
		}
	}
}

func HandleClientMessage(hub *game.GameHub, c *types.Client, raw []byte) {
	var base types.BaseMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		log.Printf("invalid message from %s: %v", c.UUID, err)
		return
	}

	switch base.Type {
	case "playerInput":
		var moveMsg types.PlayerInputMessage
		if err := json.Unmarshal(raw, &moveMsg); err != nil {
			log.Printf("bad move message: %v", err)
			return
		}
		game.UpdatePlayerMovement(hub, c, &moveMsg)

	case "projectile":
		var projMsg types.ProjectileMessage
		if err := json.Unmarshal(raw, &projMsg); err != nil {
			log.Printf("bad projectile message: %v", err)
			return
		}
		hub.AddProjectileFromClient(c, &projMsg)

	default:
		log.Printf("unhandled message type: %s", base.Type)
	}
}

func HandleSocket(hub *game.GameHub) {

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrader error", err)
			return
		}

		client := &types.Client{
			UUID:     uuid.New().String(),
			Conn:     ws,
			Send:     make(chan []byte, 256),
			LastSeen: time.Now(),
		}

		log.Println("New client connected", client.UUID)

		hub.GetRegisterChan() <- client

		//   here call go  client.readPump and write pump functions
		ReadPump(hub, client)
		WritePump(client)

	})

	fmt.Println("The game server is running at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
