package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	transactionTracker "francodominguez.com/personal-workflows/app/services/transaction_tracker"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", transactionTracker.HandleWebhook)
	mux.HandleFunc("PUT /daily-allowance", transactionTracker.HandleDailyAllowance)
	mux.HandleFunc("GET /balance", transactionTracker.HandleBalance)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("transaction tracker listening on :%s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
