# Multiplayer Hangman

A full-stack, Hangman game supporting multiplayer and AI gameplay. Features real-time play via WebSockets, persistent accounts and leaderboard (SQLite), and a clean, responsive UI built with Go, HTMX, and CSS.

## To play the game just visit:  
https://hangman-challenge.up.railway.app/  


## Local Setup:
```
git clone https://github.com/fernandomesser/hangman.git
```

Note: If running locally and want to use Gemini guessing, message me about the API key.  
```
export GEMINI_API_KEY=your_google_gemini_api_key  # (optional)
go run main.go
```

## Features:

- User Authentication  
- Multiplayer & AI: Human-vs-Human (live WebSocket games) and Human-vs-AI (Gemini AI-powered opponent with fallback frequency-based guessing).  
- Real-Time Gameplay: All guesses sync instantly for both players using WebSockets.  
- Game Hints: Each game allows a single hint—reveals a letter, powered by backend logic.  
- Leaderboard: Tracks user wins and "best score" (fewest incorrect guesses).  
- Mobile-First UI: CSS designed for phone or desktop.
