package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartProfilerWritesExecutionTraceOnStop(t *testing.T) {
	tempDir := t.TempDir()
	tracePath := filepath.Join(tempDir, "editor.trace")

	profiler, err := startProfiler(profilerConfig{TracePath: tracePath})
	if err != nil {
		t.Fatalf("startProfiler() error = %v", err)
	}

	if profiler.traceFile == nil {
		t.Fatal("expected trace file to be opened")
	}

	if err := profiler.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	info, err := os.Stat(tracePath)
	if err != nil {
		t.Fatalf("stat trace file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected execution trace file to be non-empty")
	}
	if profiler.traceFile != nil {
		t.Fatal("expected trace file handle to be cleared after Stop")
	}
}
