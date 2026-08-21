// Command task140-reliability serves the reliability analysis & safety
// assessment HTTP API backed by SQLite, and provides a --smoke-test that
// exercises the full contract (component library, FTA cut sets & probability,
// FMEA RPN & severity floor, RBD reduction, failure-data fit, analysis
// lifecycle, restart recovery, frontend page) without real-time sleeps.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"task140-reliability/internal/httpapi"
	"task140-reliability/internal/selfcheck"
	"task140-reliability/internal/service"
	"task140-reliability/internal/store"
	"task140-reliability/internal/webfs"
)

// webFS is the embedded static frontend (native HTML/CSS/JS, no build step).
var webFS = http.FS(webfs.FS())

func main() {
	smoke := flag.Bool("smoke-test", false, "run self-check and exit")
	migrateOnly := flag.Bool("migrate-only", false, "apply schema and exit")
	dbPath := flag.String("db", "reliability.db", "SQLite database file path")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	if *smoke {
		if err := selfcheck.Run(); err != nil {
			fmt.Println("smoke-test: FAIL:", err)
			osExit(1)
		}
		fmt.Println("smoke-test: ok")
		osExit(0)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	// Restart-recovery: recompute every derived result from the persisted
	// authoritative inputs so a process that died between writes converges to
	// the same state.
	if n, err := svc.ReconcileAll(context.Background()); err != nil {
		log.Printf("reconcile on startup: %v", err)
	} else if n > 0 {
		log.Printf("reconciled %d analyses on startup", n)
	}

	if *migrateOnly {
		fmt.Println("migrate-only: schema applied")
		return
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpapi.NewMux(httpapi.Services{Svc: svc}, webFS),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("reliability %s listening on %s (db=%s)", httpapi.Version, *addr, *dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// osExit is indirected so tests can substitute it; in production it is os.Exit.
var osExit = os.Exit
