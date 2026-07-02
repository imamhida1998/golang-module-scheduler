// Contoh membaca ekspresi dari environment (SCHEDULE_EXPRESSION) via godotenv.
//
// Siapkan file .env (lihat .env.example), lalu jalankan:
//
//	go run ./examples/env
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/funxdofficial/golang-module-scheduler/cron"
)

func main() {
	sched, err := cron.NewFromEnv(func() {
		log.Println("Menjalankan job terjadwal…")
	})
	if err != nil {
		log.Fatalf("gagal memuat jadwal: %v", err)
	}

	expr := sched.Expression()
	log.Printf("Zona waktu : %s", cron.Location())
	log.Printf("Ekspresi   : %s (%s)", expr, expr.Mode())
	log.Printf("Jadwal     : %s", expr.Describe())

	if err := sched.Start(); err != nil {
		log.Fatal(err)
	}
	defer sched.Stop()

	// Tunggu sinyal shutdown (Ctrl+C) agar job ter-flush dengan rapi.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Menghentikan scheduler…")
}
