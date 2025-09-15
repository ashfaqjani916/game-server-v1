package main

import (
	"game-server-v1/pkg/game"
	"game-server-v1/pkg/network"
)

func main() {

	hub := game.NewGameHub(nil)

	hub.Start()

	network.HandleSocket(hub)
}
