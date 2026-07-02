package cron

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

// DefaultTimezone adalah zona waktu default bila tidak di-override lewat env.
const DefaultTimezone = "Asia/Jakarta"

// EnvExpression adalah nama env var untuk Ekspresi Jadwal.
const EnvExpression = "SCHEDULE_EXPRESSION"

// Location mengembalikan zona waktu yang dipakai scheduler.
//
// Urutan prioritas:
//  1. env SCHEDULE_TZ
//  2. env TZ
//  3. DefaultTimezone ("Asia/Jakarta")
//
// Bila database zona waktu tidak tersedia di sistem, fallback ke offset
// tetap WIB (UTC+7).
func Location() *time.Location {
	tz := os.Getenv("SCHEDULE_TZ")
	if tz == "" {
		tz = os.Getenv("TZ")
	}
	if tz == "" {
		tz = DefaultTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

// FromEnv membaca file .env (jika ada) lalu mem-parse ekspresi dari
// SCHEDULE_EXPRESSION. godotenv.Load bersifat opsional — bila file .env
// tidak ada, env dari sistem tetap dipakai.
func FromEnv() (*Expression, error) {
	_ = godotenv.Load()
	return Parse(os.Getenv(EnvExpression))
}

// NewFromEnv adalah pintasan: baca ekspresi dari env, lalu buat Scheduler
// dengan zona waktu dari Location().
func NewFromEnv(job func()) (*Scheduler, error) {
	expr, err := FromEnv()
	if err != nil {
		return nil, err
	}
	return NewWithLocation(expr, job, Location()), nil
}
