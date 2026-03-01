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
	"canoe-slalom-live/store"
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
		"penaltyClass": func(r *store.RunResult) string {
			if r == nil {
				return ""
			}
			if r.PenaltyMisses > 0 {
				return "penalty-miss"
			}
			if r.PenaltyTouches > 0 {
				return "penalty-touch"
			}
			return "penalty-clean"
		},
		"sparkRawPct": func(v interface{}) int {
			switch r := v.(type) {
			case *store.RunResult:
				if r == nil || r.TotalTimeMs <= 0 {
					return 100
				}
				pct := (r.RawTimeMs * 100) / r.TotalTimeMs
				if pct > 100 {
					pct = 100
				}
				if pct < 1 {
					pct = 1
				}
				return pct
			case *store.LatestRunDetail:
				if r == nil || r.TotalTimeMs <= 0 {
					return 100
				}
				pct := (r.RawTimeMs * 100) / r.TotalTimeMs
				if pct > 100 {
					pct = 100
				}
				if pct < 1 {
					pct = 1
				}
				return pct
			default:
				return 100
			}
		},
		"sub": func(a, b int) int { return a - b },
		"sparkPenPct": func(v interface{}) int {
			switch r := v.(type) {
			case *store.RunResult:
				if r == nil || r.TotalTimeMs <= 0 || r.PenaltySeconds <= 0 {
					return 0
				}
				penMs := r.PenaltySeconds * 1000
				pct := (penMs * 100) / r.TotalTimeMs
				if pct < 1 {
					pct = 1
				}
				if pct > 99 {
					pct = 99
				}
				return pct
			case *store.LatestRunDetail:
				if r == nil || r.TotalTimeMs <= 0 || r.PenaltySeconds <= 0 {
					return 0
				}
				penMs := r.PenaltySeconds * 1000
				pct := (penMs * 100) / r.TotalTimeMs
				if pct < 1 {
					pct = 1
				}
				if pct > 99 {
					pct = 99
				}
				return pct
			default:
				return 0
			}
		},
	}

	// Parse templates
	tmpls := map[string]*template.Template{
		"event":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/event.html")),
		"athlete":             template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/athlete.html")),
		"judge":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/judge_run.html")),
		"judge_edit":          template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/judge_edit_run.html")),
		"leaderboard":         template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/leaderboard.html", "templates/leaderboard_partial.html")),
		"leaderboard_partial": template.Must(template.New("leaderboard_partial.html").Funcs(funcMap).ParseFiles("templates/leaderboard_partial.html")),
		"gallery":             template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/gallery.html")),
		"compare":             template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/compare.html")),
		"commentator":         template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/commentator.html", "templates/commentator_partial.html")),
		"commentator_partial": template.Must(template.New("commentator_partial.html").Funcs(funcMap).ParseFiles("templates/commentator_partial.html")),
		"error":               template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles("templates/layout.html", "templates/error.html")),
	}

	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		log.Println("WARNING: ADMIN_TOKEN not set, auth disabled for judge/admin routes")
	}

	deps := &handler.Deps{
		DB:         database,
		Tmpls:      tmpls,
		AdminToken: adminToken,
		Sessions:   handler.NewSessionStore(),
	}

	mux := http.NewServeMux()

	// Favicon and robots.txt
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y=".9em" font-size="90">🛶</text></svg>`))
	})
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nAllow: /\n"))
	})

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Public routes
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/events/demo-slalom-2026", http.StatusFound)
	})
	mux.HandleFunc("GET /events/{slug}", deps.EventPage)
	mux.HandleFunc("GET /events/{slug}/leaderboard", deps.LeaderboardPage)
	mux.HandleFunc("GET /events/{slug}/photos", deps.GalleryPage)
	mux.HandleFunc("GET /events/{slug}/commentator", deps.CommentatorPage)
	mux.HandleFunc("GET /events/{slug}/compare", deps.ComparePage)
	mux.HandleFunc("GET /events/{slug}/athletes/{id}", deps.AthletePage)

	// Judge routes (protected by admin token auth)
	mux.HandleFunc("GET /judge/events/{slug}", deps.RequireAuth(deps.JudgePage))
	mux.HandleFunc("POST /judge/events/{slug}/runs", deps.RequireAuth(deps.SubmitRun))
	mux.HandleFunc("GET /judge/events/{slug}/runs/{id}/edit", deps.RequireAuth(deps.EditRunPage))
	mux.HandleFunc("POST /judge/events/{slug}/runs/{id}", deps.RequireAuth(deps.UpdateRunHandler))
	mux.HandleFunc("POST /judge/events/{slug}/runs/{id}/delete", deps.RequireAuth(deps.DeleteRunHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler.SecurityHeaders(handler.LoggingMiddleware(mux)),
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
	if adminToken != "" {
		log.Printf("Judge panel (with token): http://localhost:%s/judge/events/demo-slalom-2026?token=%s", port, adminToken)
	} else {
		log.Printf("Judge panel (no auth): http://localhost:%s/judge/events/demo-slalom-2026", port)
	}
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	fmt.Println("Server stopped.")
}
