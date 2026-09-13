//go:build !windows

package logger

import (
	"bytes"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// SetLogToFile redirects stderr and stdout to log files in /data/logs/{alias}/.
// Creates separate .err and .log files using the provided alias.
// Uses O_APPEND for multi-process safety.
func SetLogToFile(path, alias string) {
	if alias == "" {
		panic("SetLogToFile called with empty alias")
	}

	dir := filepath.Join(path, alias)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = alias
	}
	errorFileName := filepath.Join(dir, alias+".err")
	logFileName := filepath.Join(dir, alias+".log")

	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY

	errorFile, err := os.OpenFile(errorFileName, flags, 0644)
	if err != nil {
		log.Fatalf("Failed to create error file: %v", err)
	}

	logFile, err := os.OpenFile(logFileName, flags, 0644)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}

	if err := unix.Dup2(int(errorFile.Fd()), int(os.Stderr.Fd())); err != nil {
		log.Fatalf("Failed to redirect stderr: %v", err)
	}
	errorFile.Close()

	if err := unix.Dup2(int(logFile.Fd()), int(os.Stdout.Fd())); err != nil {
		log.Fatalf("Failed to redirect stdout: %v", err)
	}
	logFile.Close()
}

// DumpPprofToFile writes heap profile data to /data/logs/{alias}/{alias}.dat.
// Creates the directory if it doesn't exist.
func DumpPprofToFile(path, alias string) error {
	if alias == "" {
		return os.ErrInvalid
	}

	dir := filepath.Join(path, alias)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	datFileName := filepath.Join(dir, alias+".dat")

	datFile, err := os.Create(datFileName)
	if err != nil {
		return err
	}
	defer datFile.Close()

	if err := pprof.WriteHeapProfile(datFile); err != nil {
		return err
	}

	return nil
}

// DumpPprofToBytes collects a 5-second CPU profile and a heap snapshot,
// returning both as separate byte slices. This call blocks for 5 seconds.
func DumpPprofToBytes() (heap []byte, cpu []byte, err error) {
	var cpuBuf bytes.Buffer
	if err = pprof.StartCPUProfile(&cpuBuf); err != nil {
		return nil, nil, err
	}
	time.Sleep(5 * time.Second)
	pprof.StopCPUProfile()

	var heapBuf bytes.Buffer
	if err = pprof.WriteHeapProfile(&heapBuf); err != nil {
		return nil, cpuBuf.Bytes(), err
	}
	return heapBuf.Bytes(), cpuBuf.Bytes(), nil
}
