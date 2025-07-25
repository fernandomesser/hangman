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

// Represents a single WebSocket client connection.
// 'role' distinguishes between player 1/2 (or watcher, if extended).
type Client struct {
	conn *websocket.Conn
	role string // "1" or "2"
}

var (
	// Global registry: for every gameID, holds all currently connected clients to that game.
	clients   = make(map[string][]*Client)
	clientsMu sync.Mutex // Guards all access to the above map.

	// Allows WebSocket upgrade; insecurely allows *any* origin. Secure in dev, dangerous in prod.
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Format for all WebSocket messages (sent and received).
type WSMessage struct {
	GameID  string      `json:"game_id"`
	Action  string      `json:"action"`  // e.g. "state", "guess", "error"
	Player  string      `json:"player"`  // optional
	Payload string      `json:"payload"` // e.g. letter guessed
	State   interface{} `json:"state"`   // embedded rendered game state per client
}

// ----------- WebSocket Handler ----------- //

// HTTP handler: Upgrade connection to WebSocket and process game messages.
// Path: /ws
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the user is authenticated (has cookie)
	userCookie, err := r.Cookie("user")
	if err != nil || userCookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Require game context
	gameCookie, err := r.Cookie("game_id")
	if err != nil || gameCookie.Value == "" {
		http.Error(w, "Missing game ID", http.StatusBadRequest)
		return
	}
	// Player role ("1", "2"); fallback "" if missing
	roleCookie, err := r.Cookie("role")
	role := ""
	if err == nil {
		role = roleCookie.Value
	}
	gameID := gameCookie.Value

	// Upgrade HTTP conn to WebSocket (handshake/protocol switch)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}

	// Create tracked client struct with credentials
	client := &Client{
		conn: conn,
		role: role,
	}

	// Register this client connection with its game, thread-safe.
	clientsMu.Lock()
	clients[gameID] = append(clients[gameID], client)
	clientsMu.Unlock()

	// On function exit (client disconnect or handler exit), remove this client.
	defer func() {
		clientsMu.Lock()
		// Remove from slice of clients for this game, if found
		clientsForGame := clients[gameID]
		for i, c := range clientsForGame {
			if c == client {
				clients[gameID] = append(clientsForGame[:i], clientsForGame[i+1:]...)
				break
			}
		}
		// If this was the last client for the game, remove entry to prevent memory leak
		if len(clients[gameID]) == 0 {
			delete(clients, gameID)
		}
		clientsMu.Unlock()
		conn.Close()
	}()

	// Main receive loop: wait for client messages
	for {
		_, msgBytes, err := client.conn.ReadMessage()
		if err != nil {
			break // client disconnected or error
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			fmt.Println("Invalid WS message:", err)
			continue
		}

		// Core game logic: handle "guess" action only
		if msg.Action == "guess" {
			// Use gameID/role for *this* connection (safer/more robust than trusting payload)
			gameID := gameCookie.Value
			game := games[gameID]
			if game == nil || game.Status == "finished" {
				continue // game not found or over
			}

			// Validate guess: must be a single a-z letter
			letter := strings.ToLower(strings.TrimSpace(msg.Payload))
			if len(letter) != 1 || letter < "a" || letter > "z" {
				continue // ignore invalid
			}

			// If already guessed, send error message (privately, do NOT broadcast).
			if game.GuessedLetters[letter] {
				errMsg := WSMessage{
					GameID:  game.ID,
					Action:  "error",
					Payload: fmt.Sprintf("Letter '%s' has already been guessed.", letter),
				}
				data, _ := json.Marshal(errMsg)
				client.conn.WriteMessage(websocket.TextMessage, data)
				continue
			}

			// Register the guess (update game state accordingly)
			logic.RegisterGuess(game, letter)

			// If single-player vs AI and it's now computer's turn: have AI make a move
			if game.Status != "finished" && game.Player2 == "Computer" && game.PlayerTurn == 2 {
				aiGuess := logic.AIGuess(game)
				logic.RegisterGuess(game, aiGuess)
			}

			// When the game ends (win/loss), update leaderboard IF not a draw
			if game.Status == "finished" && game.Winner != "Draw" {
				updateLeaderboard(game.Winner, game.IncorrectGuesses) // (see other files)
			}

			// Broadcast updated state to *all* clients for this game
			BroadcastToClients(WSMessage{
				GameID: game.ID,
				Action: "state",
			})
		}
	} // end for loop
}

// Broadcast a message (with game state) to every WebSocket client for the game.
// Each client gets their own view of state (depends on their role).
func BroadcastToClients(msg WSMessage) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	gameID := msg.GameID

	clientsForGame, ok := clients[gameID]
	if !ok {
		return
	}

	for _, client := range clientsForGame {
		// Re-build state for each client's view of game (role-sensitive)
		stateForClient := buildGameState(games[gameID], client.role)
		msg.State = stateForClient

		// JSON encode and write over WebSocket
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling WSMessage:", err)
			continue
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Println("Error writing WS message, closing conn:", err)
			client.conn.Close()
			// Remove this client from the list (prevent leaking dead conns)
			for i, c := range clientsForGame {
				if c == client {
					clients[gameID] = append(clientsForGame[:i], clientsForGame[i+1:]...)
					break
				}
			}
		}
	}
}

// Goroutine starter: listens on global wsBroadcast chan and rebroadcasts messages as needed.
// Allows other goroutines/files to trigger a broadcast by sending to wsBroadcast.
func StartWSBroadcaster() {
	go func() {
		for {
			msg := <-wsBroadcast
			BroadcastToClients(msg)
		}
	}()
}
