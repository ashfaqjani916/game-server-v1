package types

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	UUID     string          `json:"uuid"`
	Conn     *websocket.Conn `json:"-"`
	Send     chan []byte     `json:"-"`
	Player   *Player         `json:"player"`
	LastSeen time.Time       `json:"lastSeen"`
}

// Player represents a game player with position and state
type Player struct {
	ID         string    `json:"id"`
	PosX       float64   `json:"posX"`
	PosY       float64   `json:"posY"`
	MoveX      float64   `json:"moveX"`
	MoveY      float64   `json:"moveY"`
	FacingLeft bool      `json:"facingLeft"`
	MoveSpeed  float64   `json:"moveSpeed"`
	LastUpdate time.Time `json:"lastUpdate"`
	Health     int       `json:"health"`
	MaxHealth  int       `json:"maxHealth"`
	IsAlive    bool      `json:"isAlive"`
}

type Projectile struct {
	ID        string        `json:"id"`      // unique identifier
	OwnerID   string        `json:"ownerId"` // player who fired it
	PosX      float64       `json:"posX"`
	PosY      float64       `json:"posY"`
	VelX      float64       `json:"velX"`
	VelY      float64       `json:"velY"`
	CreatedAt time.Time     `json:"createdAt"`
	Lifetime  time.Duration `json:"lifetime"` // remaining lifetime
	Radius    float64       `json:"radius"`   // hitbox radius
	Damage    int           `json:"damage"`   // how much damage it deals
}

// Message types for client-server communication
type MessageType string

const (
	PlayerInputMsg  MessageType = "playerInput"
	PlayerStateMsg  MessageType = "playerState"
	PlayerIDMsg     MessageType = "playerId"
	PlayerJoinedMsg MessageType = "playerJoined"
	PlayerLeftMsg   MessageType = "playerLeft"
	GameStateMsg    MessageType = "gameState"
	ChatMsg         MessageType = "chat"
	ErrorMsg        MessageType = "error"
)

// BaseMessage is the common wrapper for all messages
type BaseMessage struct {
	Type string `json:"type"`
}

// ProjectileMessage is sent by the client when firing
type ProjectileMessage struct {
	Type         string  `json:"type"`
	PlayerID     string  `json:"playerId"`
	PosX         float64 `json:"posX"`
	PosY         float64 `json:"posY"`
	DirX         float64 `json:"dirX"`
	DirY         float64 `json:"dirY"`
	Speed        float64 `json:"speed"`
	ProjectileID string  `json:"projectileId"`
}

// PlayerInputMessage represents input from client
type PlayerInputMessage struct {
	Type       string  `json:"type"`
	PlayerID   string  `json:"playerId"`
	MoveX      float64 `json:"moveX"`
	MoveY      float64 `json:"moveY"`
	FacingLeft bool    `json:"facingLeft"`
	Timestamp  float64 `json:"timestamp"`
	SequenceID int64   `json:"sequenceId"`
}

// PlayerStateMessage represents server-authoritative player state
type PlayerStateMessage struct {
	Type       string  `json:"type"`
	PlayerID   string  `json:"playerId"`
	PosX       float64 `json:"posX"`
	PosY       float64 `json:"posY"`
	MoveX      float64 `json:"moveX"`
	MoveY      float64 `json:"moveY"`
	FacingLeft bool    `json:"facingLeft"`
	Timestamp  float64 `json:"timestamp"`
	SequenceID int64   `json:"sequenceId"`
}

// PlayerIDMessage sent to client upon connection
type PlayerIDMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
}

// PlayerJoinedMessage broadcast when a player joins
type PlayerJoinedMessage struct {
	Type     string  `json:"type"`
	PlayerID string  `json:"playerId"`
	PosX     float64 `json:"posX"`
	PosY     float64 `json:"posY"`
}

// PlayerLeftMessage broadcast when a player leaves
type PlayerLeftMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
}

// GameStateMessage contains full game state
type GameStateMessage struct {
	Type      string             `json:"type"`
	Players   map[string]*Player `json:"players"`
	Timestamp float64            `json:"timestamp"`
}

// ChatMessage for player communication
type ChatMessage struct {
	Type      string  `json:"type"`
	PlayerID  string  `json:"playerId"`
	Message   string  `json:"message"`
	Timestamp float64 `json:"timestamp"`
}

// ErrorMessage for error communication
type ErrorMessage struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ClientAction represents actions that can be performed on clients
type ClientAction struct {
	Type   string
	Client *Client
	Data   interface{}
}

// GameConfig holds game configuration
type GameConfig struct {
	TickRate     int           `json:"tickRate"`
	TickInterval time.Duration `json:"tickInterval"`
	MoveSpeed    float64       `json:"moveSpeed"`
	WorldBounds  WorldBounds   `json:"worldBounds"`
	MaxPlayers   int           `json:"maxPlayers"`
}

// WorldBounds defines the game world boundaries
type WorldBounds struct {
	MaxX float64 `json:"maxX"`
	MaxY float64 `json:"maxY"`
	MinX float64 `json:"minX"`
	MinY float64 `json:"minY"`
}

// Constants for game configuration
const (
	DefaultTickRate     = 30
	DefaultMoveSpeed    = 5.0
	DefaultMaxPlayers   = 100
	DefaultMaxX         = 50.0
	DefaultMaxY         = 50.0
	DefaultMinX         = -50.0
	DefaultMinY         = -50.0
	DefaultPlayerHealth = 100

	// Connection timeouts
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = (PongWait * 9) / 10
	MaxMessageSize = 512

	// Client timeouts
	ClientTimeout = 30 * time.Second
)

// GetDefaultConfig returns default game configuration
func GetDefaultConfig() *GameConfig {
	return &GameConfig{
		TickRate:     DefaultTickRate,
		TickInterval: time.Second / DefaultTickRate,
		MoveSpeed:    DefaultMoveSpeed,
		WorldBounds: WorldBounds{
			MaxX: DefaultMaxX,
			MaxY: DefaultMaxY,
			MinX: DefaultMinX,
			MinY: DefaultMinY,
		},
		MaxPlayers: DefaultMaxPlayers,
	}
}

// NewPlayer creates a new player with default values
func NewPlayer(id string) *Player {
	return &Player{
		ID:         id,
		PosX:       0,
		PosY:       0,
		MoveX:      0,
		MoveY:      0,
		FacingLeft: false,
		MoveSpeed:  DefaultMoveSpeed,
		LastUpdate: time.Now(),
		Health:     DefaultPlayerHealth,
		MaxHealth:  DefaultPlayerHealth,
		IsAlive:    true,
	}
}

// IsValidMovement checks if movement input is within acceptable bounds
func (p *PlayerInputMessage) IsValidMovement() bool {
	return p.MoveX >= -1.0 && p.MoveX <= 1.0 && p.MoveY >= -1.0 && p.MoveY <= 1.0
}

// ClampPosition ensures position is within world bounds
func (wb *WorldBounds) ClampPosition(x, y float64) (float64, float64) {
	if x < wb.MinX {
		x = wb.MinX
	} else if x > wb.MaxX {
		x = wb.MaxX
	}

	if y < wb.MinY {
		y = wb.MinY
	} else if y > wb.MaxY {
		y = wb.MaxY
	}

	return x, y
}
