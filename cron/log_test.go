package cron

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	SetLogOutput(io.Discard)
	DisableColor()
	os.Exit(m.Run())
}

func TestLogErrorFormat(t *testing.T) {
	var buf bytes.Buffer
	SetLogOutput(&buf)
	DisableColor()

	logError("expression nil")

	out := buf.String()
	if !strings.Contains(out, "[ERROR]") {
		t.Fatalf("harus ada [ERROR], dapat: %q", out)
	}
	if !strings.Contains(out, "log_test.go:") {
		t.Fatalf("harus ada file:line, dapat: %q", out)
	}
	if !strings.Contains(out, "expression nil") {
		t.Fatalf("harus ada pesan error, dapat: %q", out)
	}
}

func TestLogInfoAndSuccess(t *testing.T) {
	var buf bytes.Buffer
	SetLogOutput(&buf)
	DisableColor()

	logInfo("[START]", "scheduler -> Ekspresi: 00-00-0000 00-02-00 (interval)")
	logSuccess("Terjadwal: Setiap 2 menit")

	out := buf.String()
	if !strings.Contains(out, "[INFO]") || !strings.Contains(out, "[START]") {
		t.Fatalf("harus ada [INFO] [START], dapat: %q", out)
	}
	if !strings.Contains(out, "[SUCCESS]") {
		t.Fatalf("harus ada [SUCCESS], dapat: %q", out)
	}
}

func TestSchedulerStartStopLogging(t *testing.T) {
	var buf bytes.Buffer
	SetLogOutput(&buf)
	DisableColor()

	expr := MustParse("00-00-0000 00-00-01")
	s := New(expr, func() {})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.Stop()

	out := buf.String()
	for _, want := range []string{"FUNXD Schedular", "[START]", "[STOP]", "[SUCCESS]"} {
		if !strings.Contains(out, want) {
			t.Errorf("log Start/Stop harus mengandung %q, dapat: %q", want, out)
		}
	}
}

func TestSchedulerStartErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	SetLogOutput(&buf)
	DisableColor()

	s := New(nil, func() {})
	if err := s.Start(); err == nil {
		t.Fatal("harusnya error")
	}
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Fatalf("harus ada [ERROR], dapat: %q", buf.String())
	}
}

func TestPrintBanner(t *testing.T) {
	var buf bytes.Buffer
	SetLogOutput(&buf)
	DisableColor()

	printBanner()
	printBanner()

	out := buf.String()
	if strings.Count(out, "FUNXD Schedular") != 2 {
		t.Fatalf("banner harus selalu tampil setiap pemanggilan, dapat %d kali", strings.Count(out, "FUNXD Schedular"))
	}
}
