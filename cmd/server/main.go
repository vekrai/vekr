package main

import (
	"log"
	"net/http"
	"os"

	app "vekr/internal"
	"vekr/internal/config"
	"vekr/internal/models"
)

func main() {
	db, err := models.Connect()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	mux := app.NewRouter(app.NewApp(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	addr := ":" + port
	log.Printf("listening on %s (env=%s)", addr, config.Env())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
