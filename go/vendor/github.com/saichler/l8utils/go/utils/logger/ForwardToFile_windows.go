//go:build windows

package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

// Windows fallback: keep stdout/stderr unchanged and only ensure directory exists.
func SetLogToFile(path, alias string) {
	if alias == "" {
		return
	}
	_ = os.MkdirAll(filepath.Join(path, alias), 0755)
}

// DumpPprofToFile writes heap profile data to path/alias/alias.dat on Windows.
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
	return pprof.WriteHeapProfile(datFile)
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
