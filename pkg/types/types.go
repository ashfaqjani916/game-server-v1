package types

type Vector2 struct{
	x float32 
	y float32
}


type PlayerState struct{
	ID string 
	Pos Vector2 
	Health int 
}

type GameState struct {
    Players map[string]*PlayerState `json:"players"`
}

//var gameState = GameState{
//    Players: make(map[string]*PlayerState),
//}

type Message struct{
	Type string `json:"type"`
	Payload interface{} `json:"payload"`
}

type PlayerJoin struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Position Vector2 `json:"position"`
	Health int `json:"health"`
}

type PlayerLeave struct {
	ID string `json:"id"`
}

type PlayerMove struct {
	ID       string  `json:"id"`
	Position Vector2 `json:"position"`
	Velocity Vector2 `json:"velocity"`
}

type PlayerShoot struct {
	ID        string  `json:"id"`
	Direction Vector2 `json:"direction"`
}

type PlayerUpdate struct {
	ID       string  `json:"id"`
	Position Vector2 `json:"position"`
	Health   int     `json:"health"`
	Score    int     `json:"score"`
}

