package cron

import (
	"fmt"
	"strings"
)

// namaBulan — nama bulan Bahasa Indonesia (index 0 = Januari).
var namaBulan = []string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// Describe mengembalikan deskripsi human-readable dalam Bahasa Indonesia.
//
// Interval:
//
//	"00-00-0000 00-02-00" → "Setiap 2 menit"
//	"00-00-0000 02-00-00" → "Setiap 2 jam"
//	"00-00-0000 00-00-30" → "Setiap 30 detik"
//
// Calendar:
//
//	"02-00-0000 02-00-00" → "Tanggal 2 setiap bulan pukul 02:00"
//	"15-06-2026 09-30-00" → "15 Juni 2026 pukul 09:30"
func (e *Expression) Describe() string {
	if e.Mode() == ModeInterval {
		return e.describeInterval()
	}
	return e.describeCalendar()
}

func (e *Expression) describeInterval() string {
	var parts []string
	if e.Hour > 0 {
		parts = append(parts, fmt.Sprintf("%d jam", e.Hour))
	}
	if e.Minute > 0 {
		parts = append(parts, fmt.Sprintf("%d menit", e.Minute))
	}
	if e.Second > 0 {
		parts = append(parts, fmt.Sprintf("%d detik", e.Second))
	}
	if len(parts) == 0 {
		return "Tidak ada interval (semua field waktu nol)"
	}
	return "Setiap " + strings.Join(parts, " ")
}

func (e *Expression) describeCalendar() string {
	pukul := fmt.Sprintf("pukul %02d:%02d", e.Hour, e.Minute)
	if e.Second > 0 {
		pukul = fmt.Sprintf("pukul %02d:%02d:%02d", e.Hour, e.Minute, e.Second)
	}

	switch {
	case e.Day != 0 && e.Month != 0 && e.Year != 0:
		return fmt.Sprintf("%d %s %d %s", e.Day, namaBulan[e.Month-1], e.Year, pukul)
	case e.Day != 0 && e.Month != 0 && e.Year == 0:
		return fmt.Sprintf("Tanggal %d %s setiap tahun %s", e.Day, namaBulan[e.Month-1], pukul)
	case e.Day != 0 && e.Month == 0:
		return fmt.Sprintf("Tanggal %d setiap bulan %s", e.Day, pukul)
	case e.Day == 0 && e.Month != 0:
		return fmt.Sprintf("Setiap hari selama bulan %s %s", namaBulan[e.Month-1], pukul)
	default:
		return fmt.Sprintf("Setiap hari %s", pukul)
	}
}
