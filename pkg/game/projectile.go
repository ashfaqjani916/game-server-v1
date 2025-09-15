package game

import (
	"game-server-v1/pkg/types"
	"time"
)

type Projectile struct {
	ID        string
	PlayerID  string
	PosX      float64
	PosY      float64
	DirX      float64
	DirY      float64
	Speed     float64
	CreatedAt time.Time
}

func NewProjectile(msg types.ProjectileMessage) *Projectile {
	return &Projectile{
		ID:        msg.ProjectileID,
		PlayerID:  msg.PlayerID,
		PosX:      msg.PosX,
		PosY:      msg.PosY,
		DirX:      msg.DirX,
		DirY:      msg.DirY,
		Speed:     msg.Speed,
		CreatedAt: time.Now(),
	}
}
