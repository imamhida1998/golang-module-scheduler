// Contoh dasar: Interval Mode.
//
// Jalankan: go run ./examples/basic
package main

import (
	"log"
	"time"

	"github.com/funxdofficial/golang-module-scheduler/cron"
)

func main() {
	expr, err := cron.Parse("00-00-0000 00-02-00")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Mode     : %s", expr.Mode())     // "interval"
	log.Printf("Describe : %s", expr.Describe()) // "Setiap 2 menit"

	sched := cron.New(expr, func() {
		log.Println("Menjalankan job: sync-menu")
	})

	if err := sched.Start(); err != nil {
		log.Fatal(err)
	}
	defer sched.Stop()

	// Biarkan berjalan sebentar untuk demo.
	time.Sleep(5 * time.Minute)
}
