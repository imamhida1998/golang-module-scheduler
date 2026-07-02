package cron

import (
	"errors"
	"log"
	"sync"
	"time"
)

// ErrNilExpression dikembalikan bila Scheduler dibuat tanpa Expression.
var ErrNilExpression = errors.New("cron: expression nil")

// ErrNilJob dikembalikan bila Scheduler dibuat tanpa job callback.
var ErrNilJob = errors.New("cron: job callback nil")

// Scheduler menjalankan sebuah job sesuai Expression.
//
// Interval Mode memakai time.Ticker; Calendar Mode menghitung waktu jalan
// berikutnya (time.Until) dan tidur sampai saat itu. Scheduler aman dipakai
// dari banyak goroutine.
type Scheduler struct {
	expr *Expression
	job  func()
	loc  *time.Location

	// onError dipanggil bila job panic (opsional).
	onError func(recovered any)

	mu      sync.Mutex
	running bool
	stop    chan struct{}
	wg      sync.WaitGroup
}

// New membuat Scheduler dengan zona waktu default (Location()).
func New(expr *Expression, job func()) *Scheduler {
	return NewWithLocation(expr, job, Location())
}

// NewWithLocation membuat Scheduler dengan zona waktu tertentu.
func NewWithLocation(expr *Expression, job func(), loc *time.Location) *Scheduler {
	if loc == nil {
		loc = time.UTC
	}
	return &Scheduler{expr: expr, job: job, loc: loc}
}

// OnError memasang handler yang dipanggil ketika job panic. Mengembalikan
// receiver agar bisa di-chain.
func (s *Scheduler) OnError(fn func(recovered any)) *Scheduler {
	s.mu.Lock()
	s.onError = fn
	s.mu.Unlock()
	return s
}

// Expression mengembalikan ekspresi yang dipakai scheduler.
func (s *Scheduler) Expression() *Expression { return s.expr }

// Start memulai scheduler di background (non-blocking). Aman dipanggil
// berkali-kali; pemanggilan kedua saat sudah berjalan tidak berefek.
func (s *Scheduler) Start() error {
	if s.expr == nil {
		return ErrNilExpression
	}
	if s.job == nil {
		return ErrNilJob
	}
	if err := s.expr.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stop = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()
	return nil
}

// Stop menghentikan scheduler dan menunggu goroutine-nya selesai. Aman
// dipanggil walau scheduler belum/berhenti berjalan.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stop)
	s.mu.Unlock()

	s.wg.Wait()
}

// Running melaporkan apakah scheduler sedang berjalan.
func (s *Scheduler) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	if s.expr.Mode() == ModeInterval {
		s.runInterval()
		return
	}
	s.runCalendar()
}

func (s *Scheduler) runInterval() {
	d := s.expr.Interval()
	if d <= 0 {
		return
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.safeRun()
		}
	}
}

func (s *Scheduler) runCalendar() {
	for {
		now := time.Now().In(s.loc)
		next, ok := s.expr.NextRun(now)
		if !ok {
			return // tidak ada jadwal berikutnya (mis. one-shot yang sudah lewat)
		}

		timer := time.NewTimer(next.Sub(now))
		select {
		case <-s.stop:
			timer.Stop()
			return
		case <-timer.C:
			s.safeRun()
			if s.expr.IsOneShot() {
				return
			}
		}
	}
}

// safeRun menjalankan job dan menangkap panic agar scheduler tetap hidup.
func (s *Scheduler) safeRun() {
	defer func() {
		if r := recover(); r != nil {
			s.mu.Lock()
			h := s.onError
			s.mu.Unlock()
			if h != nil {
				h(r)
			} else {
				log.Printf("cron: job panic: %v", r)
			}
		}
	}()
	s.job()
}
