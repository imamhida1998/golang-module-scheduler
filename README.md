# Golang Module Scheduler

Module scheduler/cron job untuk Go dengan format ekspresi custom: **Schedule Expression / Ekspresi Jadwal** — `DD-MM-YYYY HH-MM-SS`. Satu string menentukan apakah job berjalan sebagai **interval berulang** atau **jadwal kalender**, lengkap dengan deskripsi human-readable Bahasa Indonesia dan zona waktu default `Asia/Jakarta`.

**Fitur Utama:**
- ✅ **Format tunggal** — `DD-MM-YYYY HH-MM-SS`, pemisah `-`, spasi antara tanggal dan waktu
- ✅ **Dua Mode** — Interval & Calendar, dideteksi otomatis dari ekspresi
- ✅ **Human-readable** — `Describe()` mengembalikan teks seperti "Setiap 2 menit"
- ✅ **Wildcard** — `00` di DD/MM = setiap hari/bulan; `0000` di YYYY = setiap tahun
- ✅ **Timezone** — default `Asia/Jakarta`, bisa di-override via env
- ✅ **Config via env** — baca `SCHEDULE_EXPRESSION` dengan godotenv
- ✅ **Aman** — thread-safe, `Start()`/`Stop()` idempoten, panic pada job ditangkap

## Instalasi

```bash
go get github.com/funxdofficial/golang-module-scheduler/cron
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/funxdofficial/golang-module-scheduler/cron"
)

func main() {
    expr, err := cron.Parse("00-00-0000 00-02-00")
    if err != nil {
        log.Fatal(err)
    }

    log.Println(expr.Mode())     // "interval"
    log.Println(expr.Describe()) // "Setiap 2 menit"

    sched := cron.New(expr, func() {
        log.Println("Menjalankan job: sync-menu")
    })
    sched.Start()
    defer sched.Stop()

    select {} // blok selamanya
}
```

## Format Ekspresi

```
DD - MM - YYYY   HH - MM - SS
│    │     │      │    │    │
│    │     │      │    │    └─ detik  (0–59)
│    │     │      │    └────── menit  (0–59)
│    │     │      └─────────── jam    (0–23)
│    │     └────────────────── tahun  (0000 = setiap tahun)
│    └──────────────────────── bulan  (00 = setiap bulan)
└───────────────────────────── tanggal (00 = setiap hari)
```

## Dua Mode

### 1. Interval Mode

Aktif jika **tanggal = `00-00-0000`**. Hanya bagian waktu `HH-MM-SS` yang dipakai sebagai interval berulang (memakai `time.Ticker`).

| Ekspresi | Describe() |
|----------|------------|
| `00-00-0000 00-02-00` | Setiap 2 menit |
| `00-00-0000 02-00-00` | Setiap 2 jam |
| `00-00-0000 00-00-30` | Setiap 30 detik |
| `00-00-0000 01-30-00` | Setiap 1 jam 30 menit |

### 2. Calendar Mode

Aktif jika ada **field tanggal non-nol** (DD, MM, atau YYYY). Ekspresi dibaca sebagai jadwal kalender; `00`/`0000` berlaku sebagai wildcard.

| Ekspresi | Describe() | Catatan |
|----------|------------|---------|
| `02-00-0000 02-00-00` | Tanggal 2 setiap bulan pukul 02:00 | berulang tiap bulan |
| `15-06-2026 09-30-00` | 15 Juni 2026 pukul 09:30 | sekali (one-shot) |

> **Perhatian:** `00-00-0000 02-00-00` adalah **Interval** (setiap 2 jam), *bukan* kalender. Calendar Mode hanya aktif bila ada field tanggal non-nol.

## Config via Environment

`.env`:
```env
SCHEDULE_EXPRESSION=00-00-0000 00-02-00
SCHEDULE_TZ=Asia/Jakarta
```

```go
sched, err := cron.NewFromEnv(func() {
    log.Println("Menjalankan job terjadwal…")
})
if err != nil {
    log.Fatal(err)
}
sched.Start()
defer sched.Stop()
```

`NewFromEnv` memanggil `godotenv.Load()` (opsional), mem-parse `SCHEDULE_EXPRESSION`, dan memakai zona waktu dari `cron.Location()`.

## Group — banyak job seperti routing API

`Group` mengelola banyak job bernama seperti *router* untuk cron: tiap job punya nama dan ekspresinya sendiri, seluruh grup di-`Start`/`Stop` sekaligus, dan mendukung **middleware** (mirip middleware pada API routing).

```go
r := cron.NewGroup()

// middleware berlaku untuk semua job
r.Use(cron.Logger())
r.Use(cron.Recover(nil))

// daftarkan job — seperti mendefinisikan rute
r.MustHandle("sync-menu",      "00-00-0000 00-02-00", syncMenu)
r.MustHandle("bersih-cache",   "00-00-0000 02-00-00", bersihCache)
r.MustHandle("laporan-harian", "00-00-0000 23-00-00", laporanHarian)
r.MustHandle("tutup-buku",     "01-00-0000 00-05-00", tutupBuku)

log.Print(r.Routes()) // cetak tabel jadwal
r.Start()
defer r.Stop()
```

`r.Routes()` mencetak tabel seperti daftar rute:

```
NAMA            MODE      EKSPRESI              JADWAL
bersih-cache    interval  00-00-0000 02-00-00   Setiap 2 jam
laporan-harian  interval  00-00-0000 23-00-00   Setiap 23 jam
sync-menu       interval  00-00-0000 00-02-00   Setiap 2 menit
tutup-buku      calendar  01-00-0000 00-05-00   Tanggal 1 setiap bulan pukul 00:05
```

### API Group

- `NewGroup() *Group` / `NewGroupWithLocation(loc) *Group`
- `(*Group) Use(mw ...Middleware) *Group` — pasang middleware (sebelum Start)
- `(*Group) Handle(name, expr string, job func()) error` — daftarkan job
- `(*Group) HandleExpr(name string, e *Expression, job func()) error`
- `(*Group) MustHandle(name, expr string, job func()) *Group` — chainable, panic bila error
- `(*Group) Start() error` / `(*Group) Stop()` — jalankan / hentikan semua
- `(*Group) Jobs() []JobInfo` · `Len() int` · `Running() bool` · `Routes() string`

### Middleware

```go
type Middleware func(info cron.JobInfo, next func())
```

Bawaan: `cron.Logger()` (catat mulai/selesai) dan `cron.Recover(onPanic)` (tangkap panic per-job). Buat sendiri dengan mudah:

```go
func Metrics() cron.Middleware {
    return func(info cron.JobInfo, next func()) {
        start := time.Now()
        next()
        log.Printf("%s selesai dalam %s", info.Name, time.Since(start))
    }
}
```

Nama job **unik** dalam satu grup; ekspresi divalidasi saat `Handle`. Menambah job setelah `Start` ditolak.

## Timezone

Default `Asia/Jakarta`. Prioritas: env `SCHEDULE_TZ` → env `TZ` → default. Bila database zona waktu tidak tersedia di sistem, fallback ke offset tetap WIB (UTC+7).

```go
loc := cron.Location()                 // *time.Location
sched := cron.NewWithLocation(expr, job, loc)
```

## API Reference

### Parsing & Ekspresi

- `Parse(s string) (*Expression, error)` — parse & validasi ekspresi
- `MustParse(s string) *Expression` — seperti Parse, panic bila error
- `(*Expression) Validate() error` — validasi rentang field
- `(*Expression) Mode() Mode` — `ModeInterval` atau `ModeCalendar`
- `(*Expression) Describe() string` — deskripsi Bahasa Indonesia
- `(*Expression) Interval() time.Duration` — durasi total (Interval Mode)
- `(*Expression) IsOneShot() bool` — true bila DD, MM, YYYY semuanya non-nol
- `(*Expression) NextRun(after time.Time) (time.Time, bool)` — waktu jalan berikutnya
- `(*Expression) String() string` — bentuk kanonik `DD-MM-YYYY HH-MM-SS`

### Scheduler

- `New(expr *Expression, job func()) *Scheduler` — zona waktu default
- `NewWithLocation(expr, job, loc) *Scheduler` — zona waktu kustom
- `NewFromEnv(job func()) (*Scheduler, error)` — dari env + godotenv
- `(*Scheduler) OnError(fn func(recovered any)) *Scheduler` — handler panic job
- `(*Scheduler) Start() error` — mulai (non-blocking)
- `(*Scheduler) Stop()` — hentikan & tunggu goroutine selesai
- `(*Scheduler) Running() bool` — status berjalan
- `(*Scheduler) Expression() *Expression` — ekspresi yang dipakai

### Config

- `Location() *time.Location` — zona waktu efektif
- `FromEnv() (*Expression, error)` — ekspresi dari `SCHEDULE_EXPRESSION`
- Konstanta: `DefaultTimezone`, `EnvExpression`

## Validasi

`Parse` menolak ekspresi yang tidak valid:

| Field | Rentang |
|-------|---------|
| DD (tanggal) | 0–31 |
| MM (bulan)   | 0–12 |
| YYYY (tahun) | ≥ 0  |
| HH (jam)     | 0–23 |
| MM (menit)   | 0–59 |
| SS (detik)   | 0–59 |

Selain itu, Interval Mode dengan semua field waktu nol (`00-00-0000 00-00-00`) ditolak karena intervalnya nol.

## Contoh

- `examples/basic` — Interval Mode sederhana
- `examples/env` — baca dari `.env` + graceful shutdown (Ctrl+C)

```bash
go run ./examples/basic
go run ./examples/env
```

## Testing

```bash
go test ./...
```

## License

MIT
