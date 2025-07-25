package handlers

import (
	"fmt"
	"net/http"
	"wordgame/db"
	"wordgame/utils"
)

// Data structure for holding a leaderboard entry as displayed in the UI
type LeaderboardEntry struct {
	Player    string // Player username
	Wins      int    // Total wins for the player
	BestScore string // Best score ("N/A" if no games won, otherwise a number as string)
}

// Handler to display the leaderboard page (top 10 players by win count, then by best score)
func LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// Query top 10 leaderboard records, joining user id to username, sorted by most wins, then lowest best_score
	rows, err := db.DB.Query(`
        SELECT u.username, l.wins, l.best_score
        FROM leaderboard l
        JOIN users u ON l.player = u.id
        ORDER BY l.wins DESC, l.best_score ASC
        LIMIT 10
    `)
	if err != nil {
		// On DB query failure, return an HTTP 500 and the DB error message
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close() // Ensure DB rows are closed to avoid leaks

	var entries []LeaderboardEntry // Will hold leaderboard data for the template

	// Read each row from the database result
	for rows.Next() {
		var username string
		var wins, best int
		// Scan row into variables; best is best_score (could be NULL/0 if no wins yet)
		if err := rows.Scan(&username, &wins, &best); err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// If best score is not set (0 or negative), display "N/A"
		score := "N/A"
		if best > 0 {
			score = fmt.Sprintf("%d", best)
		}
		// Append leaderboard entry struct to list for rendering
		entries = append(entries, LeaderboardEntry{
			Player:    username,
			Wins:      wins,
			BestScore: score,
		})
	}

	// Render the leaderboard page ("leaderboard.html"), passing the list of entries
	utils.RenderPage(w, r, "leaderboard.html", map[string]interface{}{
		"Entries": entries,
	})
}
