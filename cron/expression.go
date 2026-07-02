// Package cron menyediakan scheduler/cron job untuk Go dengan format
// custom "Schedule Expression" / Ekspresi Jadwal: "DD-MM-YYYY HH-MM-SS".
//
// Ada dua mode:
//
//   - Interval Mode  : aktif jika tanggal = "00-00-0000". Hanya bagian waktu
//     (HH-MM-SS) yang dipakai sebagai interval berulang.
//   - Calendar Mode  : aktif jika ada field tanggal (DD, MM, atau YYYY) yang
//     non-nol. Ekspresi dibaca sebagai jadwal kalender dengan wildcard.
//
// Contoh pemakaian:
//
//	expr, err := cron.Parse("00-00-0000 00-02-00")
//	expr.Mode()     // "interval"
//	expr.Describe() // "Setiap 2 menit"
//
//	sched := cron.New(expr, func() { /* job */ })
//	sched.Start()
//	defer sched.Stop()
package cron

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Mode merepresentasikan mode eksekusi sebuah Expression.
type Mode string

const (
	// ModeInterval — job berulang tiap interval (dari bagian waktu).
	ModeInterval Mode = "interval"
	// ModeCalendar — job berjalan pada jadwal kalender (dengan wildcard).
	ModeCalendar Mode = "calendar"
)

// Expression adalah hasil parse dari sebuah Ekspresi Jadwal.
//
// Field bernilai 0 berarti wildcard / tidak dipakai:
//   - Day   0 → setiap hari
//   - Month 0 → setiap bulan
//   - Year  0 → setiap tahun
type Expression struct {
	Day    int // DD   (0–31, 0 = wildcard)
	Month  int // MM   (0–12, 0 = wildcard)
	Year   int // YYYY (0    = wildcard)
	Hour   int // HH   (0–23)
	Minute int // MM   (0–59)
	Second int // SS   (0–59)

	raw string
}

// exprPattern memvalidasi bentuk umum "DD-MM-YYYY HH-MM-SS".
var exprPattern = regexp.MustCompile(`^(\d{2})-(\d{2})-(\d{4})\s+(\d{2})-(\d{2})-(\d{2})$`)

// Parse mem-parse string "DD-MM-YYYY HH-MM-SS" menjadi *Expression.
// Mengembalikan error jika format atau nilainya tidak valid.
func Parse(s string) (*Expression, error) {
	m := exprPattern.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("cron: format tidak valid %q, harus \"DD-MM-YYYY HH-MM-SS\"", s)
	}

	atoi := func(v string) int { n, _ := strconv.Atoi(v); return n }
	e := &Expression{
		Day:    atoi(m[1]),
		Month:  atoi(m[2]),
		Year:   atoi(m[3]),
		Hour:   atoi(m[4]),
		Minute: atoi(m[5]),
		Second: atoi(m[6]),
		raw:    s,
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}
	return e, nil
}

// MustParse seperti Parse tetapi panic jika terjadi error. Berguna untuk
// ekspresi konstan yang sudah dipastikan valid.
func MustParse(s string) *Expression {
	e, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return e
}

// Validate memeriksa rentang nilai tiap field.
func (e *Expression) Validate() error {
	switch {
	case e.Day < 0 || e.Day > 31:
		return fmt.Errorf("cron: DD (tanggal) harus 0–31, dapat %d", e.Day)
	case e.Month < 0 || e.Month > 12:
		return fmt.Errorf("cron: MM (bulan) harus 0–12, dapat %d", e.Month)
	case e.Year < 0:
		return fmt.Errorf("cron: YYYY (tahun) tidak boleh negatif, dapat %d", e.Year)
	case e.Hour < 0 || e.Hour > 23:
		return fmt.Errorf("cron: HH (jam) harus 0–23, dapat %d", e.Hour)
	case e.Minute < 0 || e.Minute > 59:
		return fmt.Errorf("cron: MM (menit) harus 0–59, dapat %d", e.Minute)
	case e.Second < 0 || e.Second > 59:
		return fmt.Errorf("cron: SS (detik) harus 0–59, dapat %d", e.Second)
	}

	if e.Mode() == ModeInterval && e.Interval() <= 0 {
		return fmt.Errorf("cron: interval nol — minimal satu dari HH/MM/SS harus > 0")
	}
	return nil
}

// Mode mengembalikan mode eksekusi: ModeInterval bila tanggal = 00-00-0000,
// selain itu ModeCalendar.
func (e *Expression) Mode() Mode {
	if e.Day == 0 && e.Month == 0 && e.Year == 0 {
		return ModeInterval
	}
	return ModeCalendar
}

// Interval mengembalikan total durasi bagian waktu (HH-MM-SS). Hanya
// bermakna untuk ModeInterval.
func (e *Expression) Interval() time.Duration {
	return time.Duration(e.Hour)*time.Hour +
		time.Duration(e.Minute)*time.Minute +
		time.Duration(e.Second)*time.Second
}

// IsOneShot bernilai true jika ekspresi kalender menunjuk satu waktu pasti
// (DD, MM, dan YYYY semuanya non-nol) — job berjalan sekali lalu berhenti.
func (e *Expression) IsOneShot() bool {
	return e.Mode() == ModeCalendar && e.Day != 0 && e.Month != 0 && e.Year != 0
}

// String mengembalikan ekspresi dalam bentuk kanonik "DD-MM-YYYY HH-MM-SS".
func (e *Expression) String() string {
	return fmt.Sprintf("%02d-%02d-%04d %02d-%02d-%02d",
		e.Day, e.Month, e.Year, e.Hour, e.Minute, e.Second)
}

// NextRun menghitung waktu jalan berikutnya setelah waktu after.
//
//   - Interval : after + Interval().
//   - Calendar : waktu terdekat > after yang cocok dengan field non-wildcard.
//
// ok bernilai false bila tidak ada waktu di masa depan (mis. one-shot yang
// tanggalnya sudah lewat).
func (e *Expression) NextRun(after time.Time) (next time.Time, ok bool) {
	if e.Mode() == ModeInterval {
		d := e.Interval()
		if d <= 0 {
			return time.Time{}, false
		}
		return after.Add(d), true
	}

	loc := after.Location()
	// Pindai per-hari dari tanggal after. Batas aman ~20 tahun.
	for i := 0; i < 366*20; i++ {
		d := after.AddDate(0, 0, i)

		if e.Year != 0 {
			if d.Year() > e.Year {
				break // tahun target sudah terlewati
			}
			if d.Year() != e.Year {
				continue
			}
		}
		if e.Month != 0 && int(d.Month()) != e.Month {
			continue
		}
		if e.Day != 0 && d.Day() != e.Day {
			continue
		}

		cand := time.Date(d.Year(), d.Month(), d.Day(), e.Hour, e.Minute, e.Second, 0, loc)
		if cand.After(after) {
			return cand, true
		}
	}
	return time.Time{}, false
}
