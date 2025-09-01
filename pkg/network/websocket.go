package network

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true;
	},
}

type Client struct{
	conn *websocket.Conn
	send chan []byte
}

var (
	clients = make(map[*Client]bool)
	broadcast = make(chan []byte)
)

func HandleConnections(res http.ResponseWriter, req *http.Request) {
	ws, err := upgrader.Upgrade(res,req,nil)
	if err != nil{
		log.Fatalln("websocket upgrade failed",err)
	}

	defer ws.Close()

	client := &Client{conn : ws , send: make(chan []byte)}
	 clients[client] = true;

	for {
		_,msg,err := ws.ReadMessage()
		if err != nil{
			log.Println("Client disconnected: ",err)
			delete(clients,client)
			break
		}
		broadcast <- msg
	}
}


func handleMessages() {
	for {
		msg := <- broadcast

		for client := range clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)	
				delete(clients,client)
			}
		}
	}
}


func (c *Client) writePump(){
	for msg := range c.send{
		err := c.conn.WriteMessage(websocket.TextMessage,msg)
		if err != nil {
			log.Println("Write error",err)
			c.conn.Close()
		}
	}
}


func HandleSocket() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
			return
		}

		client := &Client{conn: ws, send: make(chan []byte, 256)}
		clients[client] = true

		go client.writePump()

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				delete(clients, client)
				break
			}
			broadcast <- msg
		}
	})

	go handleMessages()

	fmt.Println("WebSocket server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
