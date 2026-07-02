package cron

import (
	"sync"
	"testing"
)

func TestGroupRegisterAndList(t *testing.T) {
	r := NewGroup()
	if err := r.Handle("a", "00-00-0000 00-02-00", func() {}); err != nil {
		t.Fatal(err)
	}
	if err := r.Handle("b", "02-00-0000 02-00-00", func() {}); err != nil {
		t.Fatal(err)
	}
	if r.Len() != 2 {
		t.Fatalf("Len=%d, mau 2", r.Len())
	}

	// nama duplikat ditolak
	if err := r.Handle("a", "00-00-0000 00-01-00", func() {}); err == nil {
		t.Error("nama duplikat harusnya error")
	}
	// ekspresi invalid ditolak
	if err := r.Handle("c", "bukan-ekspresi", func() {}); err == nil {
		t.Error("ekspresi invalid harusnya error")
	}
}

func TestGroupMiddlewareChain(t *testing.T) {
	r := NewGroup()
	var mu sync.Mutex
	var order []string

	r.Use(func(info JobInfo, next func()) {
		mu.Lock()
		order = append(order, "mw1-before")
		mu.Unlock()
		next()
		mu.Lock()
		order = append(order, "mw1-after")
		mu.Unlock()
	})
	r.Use(func(info JobInfo, next func()) {
		mu.Lock()
		order = append(order, "mw2-"+info.Name)
		mu.Unlock()
		next()
	})

	done := make(chan struct{})
	r.MustHandle("job", "00-00-0000 00-00-01", func() {
		mu.Lock()
		order = append(order, "job")
		mu.Unlock()
		select {
		case <-done:
		default:
			close(done)
		}
	})

	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	<-done
	r.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(order) < 3 || order[0] != "mw1-before" || order[1] != "mw2-job" || order[2] != "job" {
		t.Errorf("urutan middleware salah: %v", order)
	}
}

func TestGroupStartStop(t *testing.T) {
	r := NewGroup()
	r.MustHandle("x", "00-00-0000 00-00-01", func() {})
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	if !r.Running() {
		t.Error("grup harusnya running")
	}
	// menambah job saat berjalan ditolak
	if err := r.Handle("y", "00-00-0000 00-00-01", func() {}); err == nil {
		t.Error("menambah job saat running harusnya error")
	}
	r.Stop()
	if r.Running() {
		t.Error("grup harusnya berhenti")
	}
}

func TestGroupRoutes(t *testing.T) {
	r := NewGroup()
	r.MustHandle("sync-menu", "00-00-0000 00-02-00", func() {})
	out := r.Routes()
	if out == "" || out == "(tidak ada job terdaftar)" {
		t.Errorf("Routes() kosong: %q", out)
	}
}
