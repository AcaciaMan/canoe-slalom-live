package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"canoe-slalom-live/db"
	"canoe-slalom-live/domain"
	"canoe-slalom-live/handler"
)

func main() {
	seedFlag := flag.Bool("seed", false, "Seed the database with demo data")
	flag.Parse()

	database, err := db.Open("data.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if *seedFlag {
		if err := db.Seed(database); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
		fmt.Println("Database seeded successfully")
	}

	// Template functions
	funcMap := template.FuncMap{
		"formatTime": domain.FormatTime,
	}

	// Parse templates
	tmpls := map[string]*template.Template{
		"event":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/event.html")),
		"athlete":             template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/athlete.html")),
		"judge":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/judge_run.html")),
		"leaderboard":         template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/leaderboard.html", "templates/leaderboard_partial.html")),
		"leaderboard_partial": template.Must(template.New("leaderboard_partial.html").Funcs(funcMap).ParseFiles("templates/leaderboard_partial.html")),
		"error":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/error.html")),
	}

	deps := &handler.Deps{
		DB:    database,
		Tmpls: tmpls,
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Public routes
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/events/demo-slalom-2026", http.StatusFound)
	})
	mux.HandleFunc("GET /events/{slug}", deps.EventPage)
	mux.HandleFunc("GET /events/{slug}/leaderboard", deps.LeaderboardPage)
	mux.HandleFunc("GET /events/{slug}/athletes/{id}", deps.AthletePage)

	// Judge routes
	mux.HandleFunc("GET /judge/events/{slug}", deps.JudgePage)
	mux.HandleFunc("POST /judge/events/{slug}/runs", deps.SubmitRun)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown on Ctrl+C
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		database.Close()
	}()

	log.Printf("Server running at http://localhost:%s", port)
	log.Printf("Judge panel: http://localhost:%s/judge/events/demo-slalom-2026", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	fmt.Println("Server stopped.")
}
