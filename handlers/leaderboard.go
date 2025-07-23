package handlers

import (
	"fmt"
	"net/http"
	"wordgame/db"
	"wordgame/utils"
)

type LeaderboardEntry struct {
	Player    string
	Wins      int
	BestScore string
}

func LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(`
	SELECT u.username, l.wins, l.best_score
	FROM leaderboard l
	JOIN users u ON l.player = u.id
	ORDER BY l.wins DESC, l.best_score ASC
	LIMIT 10
`)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []LeaderboardEntry

	for rows.Next() {
		var username string
		var wins, best int
		if err := rows.Scan(&username, &wins, &best); err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		score := "N/A"
		if best > 0 {
			score = fmt.Sprintf("%d", best)
		}
		entries = append(entries, LeaderboardEntry{
			Player:    username,
			Wins:      wins,
			BestScore: score,
		})
	}	

	utils.RenderPage(w, r, "leaderboard.html", map[string]interface{}{
		"Entries": entries,
	})
}
