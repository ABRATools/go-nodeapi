package hostdata

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/mem"
)

func saveOriginals() (func(), error) {
	origHostname := hostnameFunc
	origReadFile := readFileFunc
	origCpuPercent := cpuPercentFunc
	origVirtualMemory := virtualMemoryFunc

	// Return a function to restore originals.
	restore := func() {
		hostnameFunc = origHostname
		readFileFunc = origReadFile
		cpuPercentFunc = origCpuPercent
		virtualMemoryFunc = origVirtualMemory
	}
	return restore, nil
}

func TestGetHostInfo_Success(t *testing.T) {
	restore, _ := saveOriginals()
	defer restore()

	// Override dependencies for a successful run.
	hostnameFunc = func() (string, error) {
		return "testhost", nil
	}
	readFileFunc = func(filename string) ([]byte, error) {
		if filename == "/etc/os-release" {
			return []byte(`NAME="TestOS"
VERSION="1.0"`), nil
		}
		return nil, errors.New("file not found")
	}
	cpuPercentFunc = func(interval time.Duration, percpu bool) ([]float64, error) {
		return []float64{25.0}, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{
			Total:       1024,
			UsedPercent: 50.0,
		}, nil
	}

	info, err := GetHostInfo()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if info.NodeID != "testhost" {
		t.Errorf("expected NodeID 'testhost', got: %s", info.NodeID)
	}
	if info.OSName != "TestOS" {
		t.Errorf("expected OSName 'TestOS', got: %s", info.OSName)
	}
	if info.OSVersion != "1.0" {
		t.Errorf("expected OSVersion '1.0', got: %s", info.OSVersion)
	}
	if info.CPUCount != runtime.NumCPU() {
		t.Errorf("expected CPUCount %d, got: %d", runtime.NumCPU(), info.CPUCount)
	}
	if info.CPUPercent != 25.0 {
		t.Errorf("expected CPUPercent 25.0, got: %f", info.CPUPercent)
	}
	if info.TotalMemory != 1024 {
		t.Errorf("expected TotalMemory 1024, got: %d", info.TotalMemory)
	}
	if info.MemPercent != 50.0 {
		t.Errorf("expected MemPercent 50.0, got: %f", info.MemPercent)
	}
}

func TestGetHostInfo_FailureHostname(t *testing.T) {
	restore, _ := saveOriginals()
	defer restore()

	hostnameFunc = func() (string, error) {
		return "", errors.New("hostname error")
	}
	_, err := GetHostInfo()
	if err == nil || !strings.Contains(err.Error(), "failed to get hostname") {
		t.Fatalf("expected hostname error, got: %v", err)
	}
}

func TestGetHostInfo_FailureOSRelease(t *testing.T) {
	restore, _ := saveOriginals()
	defer restore()

	hostnameFunc = func() (string, error) {
		return "testhost", nil
	}
	readFileFunc = func(filename string) ([]byte, error) {
		return nil, errors.New("os-release not found")
	}
	_, err := GetHostInfo()
	if err == nil || !strings.Contains(err.Error(), "failed to read /etc/os-release") {
		t.Fatalf("expected os-release error, got: %v", err)
	}
}

func TestGetHostInfo_FailureCPU(t *testing.T) {
	restore, _ := saveOriginals()
	defer restore()

	hostnameFunc = func() (string, error) {
		return "testhost", nil
	}
	readFileFunc = func(filename string) ([]byte, error) {
		return []byte(`NAME="TestOS"
VERSION="1.0"`), nil
	}
	cpuPercentFunc = func(interval time.Duration, percpu bool) ([]float64, error) {
		return nil, errors.New("cpu error")
	}
	_, err := GetHostInfo()
	if err == nil || !strings.Contains(err.Error(), "failed to get CPU usage") {
		t.Fatalf("expected CPU error, got: %v", err)
	}
}

func TestGetHostInfo_FailureMem(t *testing.T) {
	restore, _ := saveOriginals()
	defer restore()

	hostnameFunc = func() (string, error) {
		return "testhost", nil
	}
	readFileFunc = func(filename string) ([]byte, error) {
		return []byte(`NAME="TestOS"
VERSION="1.0"`), nil
	}
	cpuPercentFunc = func(interval time.Duration, percpu bool) ([]float64, error) {
		return []float64{25.0}, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return nil, errors.New("memory error")
	}
	_, err := GetHostInfo()
	if err == nil || !strings.Contains(err.Error(), "failed to get virtual memory info") {
		t.Fatalf("expected memory error, got: %v", err)
	}
}

// TestOSReleaseParsing focuses on ensuring that the parsing of /etc/os-release is done correctly.
func TestOSReleaseParsing(t *testing.T) {
	// Here we directly test the parsing logic.
	sample := `NAME="MyTestOS"
VERSION="2.0"
OTHER="ignored"`

	// Simulate the file read by overriding readFileFunc.
	restore, _ := saveOriginals()
	defer restore()

	readFileFunc = func(filename string) ([]byte, error) {
		return []byte(sample), nil
	}
	hostnameFunc = func() (string, error) {
		return "dummy", nil
	}
	cpuPercentFunc = func(interval time.Duration, percpu bool) ([]float64, error) {
		return []float64{10.0}, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{
			Total:       2048,
			UsedPercent: 75.0,
		}, nil
	}

	info, err := GetHostInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.OSName != "MyTestOS" {
		t.Errorf("expected OSName 'MyTestOS', got: %s", info.OSName)
	}
	if info.OSVersion != "2.0" {
		t.Errorf("expected OSVersion '2.0', got: %s", info.OSVersion)
	}
}
