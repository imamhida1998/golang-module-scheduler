package cron

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestParseValid(t *testing.T) {
	e, err := Parse("15-06-2026 09-30-00")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if e.Day != 15 || e.Month != 6 || e.Year != 2026 || e.Hour != 9 || e.Minute != 30 || e.Second != 0 {
		t.Fatalf("field salah: %+v", e)
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []string{
		"",
		"15-06-2026",           // tanpa waktu
		"2026-06-15 09:30:00",  // format salah
		"32-01-2026 00-00-00",  // DD > 31
		"01-13-2026 00-00-00",  // MM > 12
		"00-00-0000 24-00-00",  // HH > 23
		"00-00-0000 00-60-00",  // menit > 59
		"00-00-0000 00-00-00",  // interval nol
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("harusnya error untuk %q", c)
		}
	}
}

func TestMode(t *testing.T) {
	cases := map[string]Mode{
		"00-00-0000 00-02-00": ModeInterval,
		"00-00-0000 02-00-00": ModeInterval,
		"02-00-0000 02-00-00": ModeCalendar,
		"15-06-2026 09-30-00": ModeCalendar,
	}
	for expr, want := range cases {
		e := MustParse(expr)
		if got := e.Mode(); got != want {
			t.Errorf("%s: Mode()=%q, mau %q", expr, got, want)
		}
	}
}

func TestDescribe(t *testing.T) {
	cases := map[string]string{
		"00-00-0000 00-02-00": "Setiap 2 menit",
		"00-00-0000 02-00-00": "Setiap 2 jam",
		"00-00-0000 00-00-30": "Setiap 30 detik",
		"00-00-0000 01-30-00": "Setiap 1 jam 30 menit",
		"02-00-0000 02-00-00": "Tanggal 2 setiap bulan pukul 02:00",
		"15-06-2026 09-30-00": "15 Juni 2026 pukul 09:30",
		"00-00-0000 00-00-00": "Tidak ada interval (semua field waktu nol)",
	}
	for expr, want := range cases {
		// pakai parse manual agar ekspresi interval-nol tetap bisa diuji Describe
		e := parseUnchecked(expr)
		if got := e.Describe(); got != want {
			t.Errorf("%s: Describe()=%q, mau %q", expr, got, want)
		}
	}
}

func TestInterval(t *testing.T) {
	e := MustParse("00-00-0000 01-30-15")
	want := time.Hour + 30*time.Minute + 15*time.Second
	if e.Interval() != want {
		t.Errorf("Interval()=%v, mau %v", e.Interval(), want)
	}
}

func TestNextRunInterval(t *testing.T) {
	e := MustParse("00-00-0000 00-02-00")
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	next, ok := e.NextRun(now)
	if !ok || !next.Equal(now.Add(2*time.Minute)) {
		t.Errorf("NextRun interval salah: %v ok=%v", next, ok)
	}
}

func TestNextRunCalendarMonthly(t *testing.T) {
	e := MustParse("02-00-0000 02-00-00") // tanggal 2 tiap bulan 02:00
	now := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	next, ok := e.NextRun(now)
	if !ok {
		t.Fatal("harusnya ada next run")
	}
	if next.Day() != 2 || next.Month() != time.February || next.Hour() != 2 {
		t.Errorf("NextRun monthly salah: %v", next)
	}
}

func TestNextRunOneShotPast(t *testing.T) {
	e := MustParse("01-01-2020 00-00-00")
	if _, ok := e.NextRun(time.Now()); ok {
		t.Error("one-shot lampau harusnya ok=false")
	}
}

func TestSchedulerInterval(t *testing.T) {
	one := MustParse("00-00-0000 00-00-01") // setiap 1 detik

	var count int32
	s := New(one, func() { atomic.AddInt32(&count, 1) })
	if err := s.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	time.Sleep(2500 * time.Millisecond)
	s.Stop()

	if got := atomic.LoadInt32(&count); got < 2 {
		t.Errorf("job jalan %d kali, harusnya >= 2", got)
	}
	if s.Running() {
		t.Error("scheduler masih running setelah Stop")
	}
}

func TestStopIdempotent(t *testing.T) {
	s := New(MustParse("00-00-0000 00-00-01"), func() {})
	s.Stop() // belum start
	_ = s.Start()
	s.Stop()
	s.Stop() // dobel
}

// parseUnchecked mem-parse tanpa Validate — hanya untuk pengujian Describe
// pada ekspresi yang sengaja "interval nol".
func parseUnchecked(s string) *Expression {
	m := exprPattern.FindStringSubmatch(s)
	atoi := func(v string) int { n := 0; for _, c := range v { n = n*10 + int(c-'0') }; return n }
	return &Expression{
		Day: atoi(m[1]), Month: atoi(m[2]), Year: atoi(m[3]),
		Hour: atoi(m[4]), Minute: atoi(m[5]), Second: atoi(m[6]), raw: s,
	}
}
