package game

import (
	"encoding/json"
	"game-server-v1/pkg/types"
	"log"
	"sync"
	"time"
)

// GameState holds the authoritative state of the game world
type GameState struct {
	Players     map[string]*types.Player     // PlayerID → Player state
	Projectiles map[string]*types.Projectile // ProjectileID → Projectile state
	LastUpdate  time.Time
	mu          sync.RWMutex
}

// GameHub is the central coordinator for all game operations
type GameHub struct {
	// Client management
	clients    map[*types.Client]bool
	clientsMux sync.RWMutex

	// Players management (legacy, still used for lookup)
	players    map[string]*types.Player
	playersMux sync.RWMutex

	//projectiles
	Projectiles map[string]*Projectile // add projectile state

	// Channels for client lifecycle
	register   chan *types.Client
	unregister chan *types.Client

	// Channels for message handling
	broadcast    chan []byte
	playerInput  chan *types.PlayerInputMessage
	clientAction chan *types.ClientAction

	// Channel for GameState updates
	gameStateUpdate chan *GameState

	// Game configuration
	config *types.GameConfig

	// Game state
	isRunning bool
	startTime time.Time
	state     *GameState

	// Statistics
	stats *GameStats
}

// GameStats holds server statistics
type GameStats struct {
	TotalConnections int64         `json:"totalConnections"`
	ActivePlayers    int           `json:"activePlayers"`
	MessagesPerSec   float64       `json:"messagesPerSecond"`
	Uptime           time.Duration `json:"uptime"`
	LastUpdate       time.Time     `json:"lastUpdate"`
	mu               sync.RWMutex
}

// NewGameHub creates and initializes a new GameHub
func NewGameHub(config *types.GameConfig) *GameHub {
	if config == nil {
		config = types.GetDefaultConfig()
	}

	return &GameHub{
		clients:         make(map[*types.Client]bool),
		players:         make(map[string]*types.Player),
		register:        make(chan *types.Client, 100),
		unregister:      make(chan *types.Client, 100),
		broadcast:       make(chan []byte, 1000),
		playerInput:     make(chan *types.PlayerInputMessage, 1000),
		clientAction:    make(chan *types.ClientAction, 500),
		gameStateUpdate: make(chan *GameState, 100), // New channel for GameState updates
		config:          config,
		isRunning:       false,
		stats:           &GameStats{LastUpdate: time.Now()},
		state: &GameState{
			Players:     make(map[string]*types.Player),
			Projectiles: make(map[string]*types.Projectile),
			LastUpdate:  time.Now(),
		},
	}
}

// Start begins the GameHub operation
func (h *GameHub) Start() {
	h.isRunning = true
	h.startTime = time.Now()

	log.Println("GameHub starting...")

	// Start the main game loop
	go h.run()

	// Start statistics updater
	go h.updateStats()

	log.Println("GameHub started successfully")
}

// Stop gracefully shuts down the GameHub
func (h *GameHub) Stop() {
	h.isRunning = false

	// Close all client connections
	h.clientsMux.Lock()
	for client := range h.clients {
		close(client.Send)
	}
	h.clientsMux.Unlock()

	log.Println("GameHub stopped")
}

// run is the main game loop
func (h *GameHub) run() {
	ticker := time.NewTicker(h.config.TickInterval)
	defer ticker.Stop()

	for h.isRunning {
		select {
		case <-ticker.C:
			h.gameTick()

		case client := <-h.register:
			h.handleClientRegister(client)

		case client := <-h.unregister:
			h.handleClientUnregister(client)

		case input := <-h.playerInput:
			h.handlePlayerInput(input)

		case message := <-h.broadcast:
			h.handleBroadcast(message)

		case action := <-h.clientAction:
			h.handleClientAction(action)

		case gameState := <-h.gameStateUpdate:
			h.handleGameStateUpdate(gameState)
		}
	}
}

// gameTick performs regular game updates - only broadcasts current state
func (h *GameHub) gameTick() {
	// Create a snapshot of the current state and broadcast to all clients
	stateCopy := h.snapshotState()
	h.broadcastGameState(stateCopy)
}

// broadcastGameState sends the GameState to all connected clients via their WritePump
func (h *GameHub) broadcastGameState(gameState *GameState) {
	// Marshal the GameState to JSON
	gameStateMsg := types.GameStateMessage{
		Type:      string(types.GameStateMsg),
		Players:   gameState.Players,
		Timestamp: float64(gameState.LastUpdate.UnixNano()) / 1e9,
	}

	data, err := json.Marshal(gameStateMsg)
	if err != nil {
		log.Printf("Error marshaling game state: %v", err)
		return
	}

	// Send to each client individually through their WritePump
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- data:
			// Successfully queued for client's WritePump
		default:
			// Client's send channel is full, disconnect them
			log.Printf("Client %s send buffer full, disconnecting", client.UUID)
			go func(c *types.Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// handleGameStateUpdate processes external GameState updates
func (h *GameHub) handleGameStateUpdate(newState *GameState) {
	// Update the hub's state with the new state
	h.state.mu.Lock()
	h.state.Players = newState.Players
	h.state.Projectiles = newState.Projectiles
	h.state.LastUpdate = newState.LastUpdate
	h.state.mu.Unlock()

	// Broadcast the updated state to all clients
	h.broadcastGameState(newState)
}

// snapshotState creates a safe copy of the current game state
func (h *GameHub) snapshotState() *GameState {
	h.state.mu.RLock()
	defer h.state.mu.RUnlock()

	playersCopy := make(map[string]*types.Player)
	for id, p := range h.state.Players {
		cp := *p
		playersCopy[id] = &cp
	}

	projectilesCopy := make(map[string]*types.Projectile)
	for id, proj := range h.state.Projectiles {
		cp := *proj
		projectilesCopy[id] = &cp
	}

	return &GameState{
		Players:     playersCopy,
		Projectiles: projectilesCopy,
		LastUpdate:  h.state.LastUpdate,
	}
}

// UpdateGameState allows external functions to update the game state
func (h *GameHub) UpdateGameState(newState *GameState) {
	select {
	case h.gameStateUpdate <- newState:
		// Successfully queued the update
	default:
		log.Println("Warning: GameState update channel is full, dropping update")
	}
}

// handleClientRegister processes new client registrations
func (h *GameHub) handleClientRegister(client *types.Client) {
	h.clientsMux.Lock()
	h.clients[client] = true
	h.clientsMux.Unlock()

	// Update statistics
	h.stats.mu.Lock()
	h.stats.TotalConnections++
	h.stats.ActivePlayers = len(h.clients)
	h.stats.mu.Unlock()

	log.Printf("Client %s registered", client.UUID)

	// Send current game state to the new client
	stateCopy := h.snapshotState()
	h.sendGameStateToClient(client, stateCopy)
}

// sendGameStateToClient sends the current GameState to a specific client
func (h *GameHub) sendGameStateToClient(client *types.Client, gameState *GameState) {
	gameStateMsg := types.GameStateMessage{
		Type:      string(types.GameStateMsg),
		Players:   gameState.Players,
		Timestamp: float64(gameState.LastUpdate.UnixNano()) / 1e9,
	}

	data, err := json.Marshal(gameStateMsg)
	if err != nil {
		log.Printf("Error marshaling game state for client %s: %v", client.UUID, err)
		return
	}

	select {
	case client.Send <- data:
		// Successfully sent
	default:
		// Client buffer full
		log.Printf("Client %s buffer full when sending initial state", client.UUID)
	}
}

// handleClientUnregister processes client disconnections
func (h *GameHub) handleClientUnregister(client *types.Client) {
	h.clientsMux.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)
	}
	h.clientsMux.Unlock()

	// Remove player if exists
	if client.Player != nil {
		h.state.mu.Lock()
		delete(h.state.Players, client.Player.ID)
		h.state.mu.Unlock()

		// Broadcast player left
		h.broadcastPlayerLeft(client.Player.ID)

		log.Printf("Player %s disconnected", client.Player.ID)
	}

	// Update statistics
	h.stats.mu.Lock()
	h.stats.ActivePlayers = len(h.clients)
	h.stats.mu.Unlock()
}

// handlePlayerInput processes player input messages
func (h *GameHub) handlePlayerInput(input *types.PlayerInputMessage) {
	// Player input is handled externally in player.go
	// This is kept for compatibility but does nothing
	log.Printf("Player input received for %s (handled externally)", input.PlayerID)
}

// handleBroadcast sends messages to all connected clients (legacy method)
func (h *GameHub) handleBroadcast(message []byte) {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			delete(h.clients, client)
			close(client.Send)
		}
	}
}

// handleClientAction processes various client actions
func (h *GameHub) handleClientAction(action *types.ClientAction) {
	switch action.Type {
	case "sendToClient":
		if data, ok := action.Data.([]byte); ok {
			select {
			case action.Client.Send <- data:
			default:
				// Client buffer full, disconnect
				h.unregister <- action.Client
			}
		}
	case "kickClient":
		h.unregister <- action.Client
	}
}

// broadcastPlayerLeft sends player left message to all clients
func (h *GameHub) broadcastPlayerLeft(playerID string) {
	msg := map[string]string{
		"type":     "playerLeft",
		"playerId": playerID,
	}
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}

// updateStats periodically updates server statistics
func (h *GameHub) updateStats() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for h.isRunning {
		<-ticker.C

		h.stats.mu.Lock()
		h.stats.Uptime = time.Since(h.startTime)
		h.stats.LastUpdate = time.Now()
		h.stats.mu.Unlock()
	}
}

// --- Getter methods ---

func (h *GameHub) GetClients() map[*types.Client]bool {
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()

	clients := make(map[*types.Client]bool)
	for client, active := range h.clients {
		clients[client] = active
	}
	return clients
}

func (h *GameHub) GetStats() GameStats {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()
	return *h.stats
}

func (h *GameHub) GetRegisterChan() chan<- *types.Client   { return h.register }
func (h *GameHub) GetUnregisterChan() chan<- *types.Client { return h.unregister }
func (h *GameHub) GetBroadcastChan() chan<- []byte         { return h.broadcast }
func (h *GameHub) GetPlayerInputChan() chan<- *types.PlayerInputMessage {
	return h.playerInput
}
func (h *GameHub) GetClientActionChan() chan<- *types.ClientAction { return h.clientAction }
func (h *GameHub) IsRunning() bool                                 { return h.isRunning }
func (h *GameHub) GetConfig() *types.GameConfig                    { return h.config }

// GetGameState returns a snapshot of the current game state
func (h *GameHub) GetGameState() *GameState {
	return h.snapshotState()
}

// GetGameStateUpdateChan returns the channel for updating game state externally
func (h *GameHub) GetGameStateUpdateChan() chan<- *GameState {
	return h.gameStateUpdate
}

func (h *GameHub) AddProjectileFromClient(c *types.Client, msg *types.ProjectileMessage) {
	projectile := NewProjectile(*msg)
	projectile.PlayerID = c.UUID // ensure it's tied to the firing client

	h.Projectiles[projectile.ID] = projectile
}
