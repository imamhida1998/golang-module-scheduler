package cron

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	ansiReset   = "\033[0m"
	ansiGray    = "\033[90m"
	ansiBlue    = "\033[1;34m"
	ansiCyan    = "\033[1;36m"
	ansiGreen   = "\033[1;32m"
	ansiRed     = "\033[1;31m"
	ansiWhite   = "\033[97m"
	ansiFunBlue = "\033[38;5;33m"
)

var (
	logMu    sync.Mutex
	logOut   io.Writer = os.Stdout
	logColor           = true
)

// SetLogOutput mengatur writer untuk output log (default os.Stdout).
func SetLogOutput(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	logMu.Lock()
	logOut = w
	logMu.Unlock()
}

// DisableColor mematikan warna ANSI pada output log.
func DisableColor() {
	logMu.Lock()
	logColor = false
	logMu.Unlock()
}

func color(code, s string) string {
	if !logColor {
		return s
	}
	return code + s + ansiReset
}

func logTimestamp() string {
	return timeNow().Format("15:04:05")
}

// timeNow dapat dioverride di test.
var timeNow = func() time.Time { return time.Now() }

func printBanner() {
	logMu.Lock()
	defer logMu.Unlock()

	b := color(ansiFunBlue, "███████╗") + color(ansiWhite, "██╗  ██╗") + color(ansiFunBlue, "██████╗ ")
	b += "\n" + color(ansiFunBlue, "██╔════╝") + color(ansiWhite, "╚██╗██╔╝") + color(ansiFunBlue, "██╔══██╗")
	b += "\n" + color(ansiFunBlue, "██║     ") + color(ansiWhite, " ╚███╔╝ ") + color(ansiFunBlue, "██║  ██║")
	b += "\n" + color(ansiFunBlue, "██║     ") + color(ansiWhite, " ██╔██╗ ") + color(ansiFunBlue, "██║  ██║")
	b += "\n" + color(ansiFunBlue, "╚██████╗") + color(ansiWhite, "██╔╝ ██╗") + color(ansiFunBlue, "██████╔╝")
	b += "\n" + color(ansiFunBlue, " ╚═════╝") + color(ansiWhite, "╚═╝  ╚═╝") + color(ansiFunBlue, "╚═════╝ ")

	fmt.Fprintln(logOut, b)
	fmt.Fprintln(logOut, color(ansiCyan, "  FUNXD Schedular"))
	fmt.Fprintln(logOut)
}

func logInfo(tag, msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Fprintf(logOut, "%s | %s | %s | %s\n",
		color(ansiGray, logTimestamp()),
		color(ansiBlue, "[INFO]"),
		color(ansiCyan, tag),
		msg,
	)
}

func logSuccess(msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Fprintf(logOut, "%s | %s -> %s\n",
		color(ansiGray, logTimestamp()),
		color(ansiGreen, "[SUCCESS]"),
		msg,
	)
}

func logError(msg string) {
	file, line := caller(2)
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Fprintf(logOut, "%s | %s | %s:%d -> %s\n",
		color(ansiGray, logTimestamp()),
		color(ansiRed, "[ERROR]"),
		file, line,
		msg,
	)
}

func caller(skip int) (file string, line int) {
	_, f, l, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}
	return filepath.Base(f), l
}
