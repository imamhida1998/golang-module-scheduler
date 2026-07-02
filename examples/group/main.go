// Contoh Group — mendaftarkan banyak job bernama seperti routing API,
// lengkap dengan middleware.
//
// Jalankan: go run ./examples/group
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/funxdofficial/golang-module-scheduler/cron"
)

func main() {
	r := cron.NewGroup()

	// Middleware berlaku untuk semua job (urut pendaftaran).
	r.Use(cron.Logger())
	r.Use(cron.Recover(nil))

	// Daftarkan job — mirip mendefinisikan rute.
	r.MustHandle("sync-menu", "00-00-0000 00-02-00", func() {
		log.Println("→ sinkronisasi menu")
	})
	r.MustHandle("bersih-cache", "00-00-0000 02-00-00", func() {
		log.Println("→ membersihkan cache")
	})
	r.MustHandle("laporan-harian", "00-00-0000 23-00-00", func() {
		log.Println("→ membuat laporan harian")
	})
	r.MustHandle("tutup-buku", "01-00-0000 00-05-00", func() {
		log.Println("→ tutup buku bulanan (tanggal 1)")
	})

	// Cetak tabel jadwal, seperti daftar rute.
	log.Printf("\n%s\n", r.Routes())

	if err := r.Start(); err != nil {
		log.Fatal(err)
	}
	defer r.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Menghentikan semua job…")
}
