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
		SELECT player, wins, best_score
		FROM leaderboard
		ORDER BY wins DESC, best_score ASC
		LIMIT 10
	`)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var user string
		var wins, best int
		rows.Scan(&user, &wins, &best)

		score := "N/A"
		if best > 0 {
			score = fmt.Sprintf("%d", best)
		}

		entries = append(entries, LeaderboardEntry{
			Player:    user,
			Wins:      wins,
			BestScore: score,
		})
	}

	utils.RenderPage(w, r, "leaderboard.html", map[string]interface{}{
		"Entries": entries,
	})
}
