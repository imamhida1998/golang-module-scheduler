package cron

import "fmt"

// Logger adalah middleware bawaan yang mencatat awal & akhir tiap job.
// Pasang lewat Group.Use.
//
//	r := cron.NewGroup()
//	r.Use(cron.Logger())
func Logger() Middleware {
	return func(info JobInfo, next func()) {
		logInfo("[RUN]", fmt.Sprintf("%s -> %s", info.Name, info.Expr.Describe()))
		next()
		logSuccess(fmt.Sprintf("%s selesai", info.Name))
	}
}

// Recover adalah middleware yang menangkap panic pada job dan meneruskannya
// ke handler onPanic (bila nil, dicatat ke log ERROR). Job lain dalam grup
// tetap berjalan.
func Recover(onPanic func(info JobInfo, recovered any)) Middleware {
	return func(info JobInfo, next func()) {
		defer func() {
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(info, r)
				}
				panic(r)
			}
		}()
		next()
	}
}
