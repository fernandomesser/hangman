package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"wordgame/logic"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	role string // "1" or "2"
}

var (
	clients   = make(map[string][]*Client) // map gameID → slice of clients
	clientsMu sync.Mutex

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// WSMessage defines the structure sent over WebSocket
type WSMessage struct {
	GameID  string      `json:"game_id"`
	Action  string      `json:"action"`  // e.g. "state"
	Player  string      `json:"player"`  // optional
	Payload string      `json:"payload"` // optional guessed letter
	State   interface{} `json:"state"`   // game state from buildGameState
}

// Handler for /ws route
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("user")
	if err != nil || userCookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	gameCookie, err := r.Cookie("game_id")
	if err != nil || gameCookie.Value == "" {
		http.Error(w, "Missing game ID", http.StatusBadRequest)
		return
	}
	roleCookie, err := r.Cookie("role")
	role := ""
	if err == nil {
		role = roleCookie.Value
	}

	gameID := gameCookie.Value

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		conn: conn,
		role: role,
	}

	clientsMu.Lock()
	clients[gameID] = append(clients[gameID], client)
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		// Remove client from slice
		clientsForGame := clients[gameID]
		for i, c := range clientsForGame {
			if c == client {
				clients[gameID] = append(clientsForGame[:i], clientsForGame[i+1:]...)
				break
			}
		}
		// If no clients left for game, delete entry
		if len(clients[gameID]) == 0 {
			delete(clients, gameID)
		}
		clientsMu.Unlock()
		conn.Close()
	}()

	for {
		_, msgBytes, err := client.conn.ReadMessage()
		if err != nil {
			break // connection closed or error
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			fmt.Println("Invalid WS message:", err)
			continue
		}

		if msg.Action == "guess" {
			// Use the gameID and role from this connection context
			gameID := gameCookie.Value
			game := games[gameID]
			if game == nil || game.Status == "finished" {
				continue
			}

			letter := strings.ToLower(strings.TrimSpace(msg.Payload))
			if len(letter) != 1 || letter < "a" || letter > "z" {
				continue // ignore invalid guesses
			}

			if game.GuessedLetters[letter] {
				// Send error only to this client (already guessed letter)
				errMsg := WSMessage{
					GameID:  game.ID,
					Action:  "error",
					Payload: fmt.Sprintf("Letter '%s' has already been guessed.", letter),
				}
				data, _ := json.Marshal(errMsg)
				client.conn.WriteMessage(websocket.TextMessage, data)
				continue
			}

			// Register the guess
			logic.RegisterGuess(game, letter)

			// AI move if vs Computer and not finished
			if game.Status != "finished" && game.Player2 == "Computer" && game.PlayerTurn == 2 {
				aiGuess := logic.AIGuess(game)
				logic.RegisterGuess(game, aiGuess)
			}

			// Update leaderboard if game ended
			if game.Status == "finished" && game.Winner != "Draw" {
				updateLeaderboard(game.Winner, game.IncorrectGuesses)
			}

			// Broadcast updated state to all clients of this game
			BroadcastToClients(WSMessage{
				GameID: game.ID,
				Action: "state",
			})
		}
	}

}

// BroadcastToClients sends the msg to all connections for msg.GameID,
// but sends a personalized buildGameState per client role.
func BroadcastToClients(msg WSMessage) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	gameID := msg.GameID

	clientsForGame, ok := clients[gameID]
	if !ok {
		return
	}

	for _, client := range clientsForGame {
		// Build game state for this client's role
		stateForClient := buildGameState(games[gameID], client.role)
		msg.State = stateForClient

		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling WSMessage:", err)
			continue
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Println("Error writing WS message, closing conn:", err)
			client.conn.Close()
			// Remove disconnected client from slice
			for i, c := range clientsForGame {
				if c == client {
					clients[gameID] = append(clientsForGame[:i], clientsForGame[i+1:]...)
					break
				}
			}
		}
	}
}

// Starts the goroutine loop to broadcast messages received on wsBroadcast channel
func StartWSBroadcaster() {
	go func() {
		for {
			msg := <-wsBroadcast
			BroadcastToClients(msg)
		}
	}()
}
