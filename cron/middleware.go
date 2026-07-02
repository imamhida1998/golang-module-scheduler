package cron

import "log"

// Logger adalah middleware bawaan yang mencatat awal & akhir tiap job ke
// standard logger. Pasang lewat Group.Use.
//
//	r := cron.NewGroup()
//	r.Use(cron.Logger())
func Logger() Middleware {
	return func(info JobInfo, next func()) {
		log.Printf("[cron] ▶ %s (%s) — %s", info.Name, info.Expr.Mode(), info.Expr.Describe())
		next()
		log.Printf("[cron] ✔ %s selesai", info.Name)
	}
}

// Recover adalah middleware yang menangkap panic pada job dan meneruskannya
// ke handler onPanic (bila nil, dicatat ke standard logger). Job lain dalam
// grup tetap berjalan.
func Recover(onPanic func(info JobInfo, recovered any)) Middleware {
	return func(info JobInfo, next func()) {
		defer func() {
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(info, r)
				} else {
					log.Printf("[cron] ✖ %s panic: %v", info.Name, r)
				}
			}
		}()
		next()
	}
}
