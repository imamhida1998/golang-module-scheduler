package cron

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// JobInfo mendeskripsikan satu job terdaftar di dalam Group.
type JobInfo struct {
	Name string
	Expr *Expression
}

// Middleware membungkus eksekusi job — mirip middleware pada API routing.
// Panggil next() untuk melanjutkan ke handler (atau middleware) berikutnya.
//
//	func Logger() cron.Middleware {
//	    return func(info cron.JobInfo, next func()) {
//	        log.Printf("▶ %s", info.Name)
//	        next()
//	        log.Printf("✔ %s selesai", info.Name)
//	    }
//	}
type Middleware func(info JobInfo, next func())

// Group adalah kumpulan job terjadwal yang dikelola bersama — seperti
// router untuk cron. Setiap job punya nama dan ekspresinya sendiri, dan
// seluruh grup di-Start / Stop sekaligus. Middleware berlaku untuk semua job.
//
// Contoh:
//
//	r := cron.NewGroup()
//	r.Use(cron.Logger())
//	r.MustHandle("sync-menu", "00-00-0000 00-02-00", syncMenu)
//	r.MustHandle("bersih-cache", "00-00-0000 02-00-00", bersihCache)
//	r.MustHandle("laporan-harian", "00-00-0000 23-00-00", laporanHarian)
//	r.Start()
//	defer r.Stop()
type Group struct {
	loc *time.Location

	mu      sync.Mutex
	mw      []Middleware
	entries []*groupEntry
	started bool
}

type groupEntry struct {
	info  JobInfo
	job   func()
	sched *Scheduler
}

// NewGroup membuat Group dengan zona waktu default (Location()).
func NewGroup() *Group {
	return NewGroupWithLocation(Location())
}

// NewGroupWithLocation membuat Group dengan zona waktu tertentu.
func NewGroupWithLocation(loc *time.Location) *Group {
	if loc == nil {
		loc = time.UTC
	}
	return &Group{loc: loc}
}

// Use mendaftarkan satu atau lebih middleware yang berlaku untuk semua job
// dalam grup. Harus dipanggil sebelum Start. Mengembalikan receiver agar
// bisa di-chain.
func (g *Group) Use(mw ...Middleware) *Group {
	g.mu.Lock()
	g.mw = append(g.mw, mw...)
	g.mu.Unlock()
	return g
}

// Handle mendaftarkan job dengan nama dan ekspresi string. Mengembalikan
// error bila ekspresi tidak valid atau nama duplikat.
func (g *Group) Handle(name, expr string, job func()) error {
	e, err := Parse(expr)
	if err != nil {
		return fmt.Errorf("cron: job %q: %w", name, err)
	}
	return g.HandleExpr(name, e, job)
}

// HandleExpr sama seperti Handle tetapi memakai *Expression yang sudah diparse.
func (g *Group) HandleExpr(name string, e *Expression, job func()) error {
	if e == nil {
		return fmt.Errorf("cron: job %q: %w", name, ErrNilExpression)
	}
	if job == nil {
		return fmt.Errorf("cron: job %q: %w", name, ErrNilJob)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.started {
		return fmt.Errorf("cron: tidak bisa menambah job %q — grup sudah berjalan", name)
	}
	for _, en := range g.entries {
		if en.info.Name == name {
			return fmt.Errorf("cron: nama job %q sudah terdaftar", name)
		}
	}
	g.entries = append(g.entries, &groupEntry{info: JobInfo{Name: name, Expr: e}, job: job})
	return nil
}

// MustHandle seperti Handle tetapi panic bila error. Cocok untuk
// pendaftaran statis, mengembalikan receiver agar bisa di-chain.
func (g *Group) MustHandle(name, expr string, job func()) *Group {
	if err := g.Handle(name, expr, job); err != nil {
		panic(err)
	}
	return g
}

// Start menjalankan seluruh job dalam grup. Bila salah satu gagal start,
// job yang sudah berjalan dihentikan kembali dan error dikembalikan.
func (g *Group) Start() error {
	printBanner()

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.started {
		return nil
	}

	for _, e := range g.entries {
		logInfo("[START]", fmt.Sprintf("%s -> Ekspresi: %s (%s)",
			e.info.Name, e.info.Expr.String(), e.info.Expr.Mode()))

		e.sched = NewWithLocation(e.info.Expr, g.wrap(e.info, e.job), g.loc)
		e.sched.label = e.info.Name
		e.sched.suppressBanner = true
		e.sched.quietLifecycle = true

		if err := e.sched.Start(); err != nil {
			logError(fmt.Sprintf("gagal start job %q: %v", e.info.Name, err))
			for _, x := range g.entries {
				if x.sched != nil {
					x.sched.Stop()
					x.sched = nil
				}
			}
			return fmt.Errorf("cron: gagal start job %q: %w", e.info.Name, err)
		}
		logSuccess(fmt.Sprintf("%s: %s", e.info.Name, e.info.Expr.Describe()))
	}
	g.started = true
	return nil
}

// Stop menghentikan seluruh job dan menunggu goroutine-nya selesai.
func (g *Group) Stop() {
	g.mu.Lock()
	entries := append([]*groupEntry(nil), g.entries...)
	g.started = false
	g.mu.Unlock()

	stopped := 0
	for _, e := range entries {
		if e.sched == nil {
			continue
		}
		wasRunning := e.sched.Running()
		e.sched.Stop()
		if wasRunning {
			logInfo("[STOP]", fmt.Sprintf("%s -> Ekspresi: %s (%s)",
				e.info.Name, e.info.Expr.String(), e.info.Expr.Mode()))
			logSuccess(fmt.Sprintf("%s berhasil dihentikan", e.info.Name))
			stopped++
		}
	}
	if stopped > 1 {
		logSuccess(fmt.Sprintf("group: %d job berhasil dihentikan", stopped))
	}
}

// Jobs mengembalikan daftar job terdaftar (salinan, urut sesuai pendaftaran).
func (g *Group) Jobs() []JobInfo {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]JobInfo, len(g.entries))
	for i, e := range g.entries {
		out[i] = e.info
	}
	return out
}

// Len mengembalikan jumlah job terdaftar.
func (g *Group) Len() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.entries)
}

// Running melaporkan apakah grup sedang berjalan.
func (g *Group) Running() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.started
}

// Routes mengembalikan tabel job yang enak dibaca — mirip daftar rute API.
//
//	NAMA            MODE      EKSPRESI              JADWAL
//	sync-menu       interval  00-00-0000 00-02-00   Setiap 2 menit
//	bersih-cache    interval  00-00-0000 02-00-00   Setiap 2 jam
func (g *Group) Routes() string {
	jobs := g.Jobs()
	if len(jobs) == 0 {
		return "(tidak ada job terdaftar)"
	}

	sorted := append([]JobInfo(nil), jobs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	nameW := len("NAMA")
	exprW := len("EKSPRESI")
	for _, j := range sorted {
		if len(j.Name) > nameW {
			nameW = len(j.Name)
		}
		if len(j.Expr.String()) > exprW {
			exprW = len(j.Expr.String())
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-*s  %-8s  %-*s  %s\n", nameW, "NAMA", "MODE", exprW, "EKSPRESI", "JADWAL")
	for _, j := range sorted {
		fmt.Fprintf(&b, "%-*s  %-8s  %-*s  %s\n",
			nameW, j.Name, j.Expr.Mode(), exprW, j.Expr.String(), j.Expr.Describe())
	}
	return strings.TrimRight(b.String(), "\n")
}

// wrap membangun rantai middleware di sekeliling job (dieksekusi urut daftar).
func (g *Group) wrap(info JobInfo, job func()) func() {
	h := job
	for i := len(g.mw) - 1; i >= 0; i-- {
		mw := g.mw[i]
		next := h
		h = func() { mw(info, next) }
	}
	return h
}
