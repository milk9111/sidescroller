package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	runtimepprof "runtime/pprof"
	runtimetrace "runtime/trace"
	"strings"
	"time"
)

type profilerConfig struct {
	PprofAddr          string
	CPUProfilePath     string
	TracePath          string
	MemProfilePath     string
	MemProfileRate     int
	MemProfileInterval time.Duration
}

type profiler struct {
	server              *http.Server
	cpuProfileFile      *os.File
	traceFile           *os.File
	memProfilePath      string
	memProfileRateReset func()
	memTicker           *time.Ticker
	memTickerDone       chan struct{}
	memSnapshotIndex    int
	stopCPUProfile      bool
	stopTrace           bool
}

func startProfiler(cfg profilerConfig) (*profiler, error) {
	p := &profiler{memProfilePath: cfg.MemProfilePath}

	if cfg.MemProfileRate > 0 {
		previousRate := runtime.MemProfileRate
		runtime.MemProfileRate = cfg.MemProfileRate
		p.memProfileRateReset = func() {
			runtime.MemProfileRate = previousRate
		}
	}

	if cfg.CPUProfilePath != "" {
		cpuFile, err := createProfileFile(cfg.CPUProfilePath)
		if err != nil {
			_ = p.Stop()
			return nil, fmt.Errorf("create cpuprofile: %w", err)
		}
		if err := runtimepprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			_ = p.Stop()
			return nil, fmt.Errorf("start cpuprofile: %w", err)
		}
		p.cpuProfileFile = cpuFile
		p.stopCPUProfile = true
		log.Printf("editor cpu profiling enabled: %s", cfg.CPUProfilePath)
	}

	if cfg.TracePath != "" {
		traceFile, err := createProfileFile(cfg.TracePath)
		if err != nil {
			_ = p.Stop()
			return nil, fmt.Errorf("create trace: %w", err)
		}
		if err := runtimetrace.Start(traceFile); err != nil {
			_ = traceFile.Close()
			_ = p.Stop()
			return nil, fmt.Errorf("start trace: %w", err)
		}
		p.traceFile = traceFile
		p.stopTrace = true
		log.Printf("editor execution trace enabled: %s", cfg.TracePath)
	}

	if cfg.PprofAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", httppprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", httppprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)
		server := &http.Server{Addr: cfg.PprofAddr, Handler: mux}
		p.server = server
		go func() {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("editor pprof server error: %v", err)
			}
		}()
		log.Printf("editor pprof listening on http://%s/debug/pprof/", cfg.PprofAddr)
	}

	if cfg.MemProfileInterval > 0 {
		if cfg.MemProfilePath == "" {
			_ = p.Stop()
			return nil, errors.New("memprofile-sample requires -memprofile")
		}
		p.memTicker = time.NewTicker(cfg.MemProfileInterval)
		p.memTickerDone = make(chan struct{})
		go p.captureMemProfiles(cfg.MemProfileInterval)
		log.Printf("editor heap snapshots enabled every %s: %s", cfg.MemProfileInterval, cfg.MemProfilePath)
	}

	return p, nil
}

func (p *profiler) Stop() error {
	var errs []error

	if p.memTicker != nil {
		p.memTicker.Stop()
		close(p.memTickerDone)
		p.memTicker = nil
	}

	if p.memProfilePath != "" {
		if err := writeHeapProfile(p.memProfilePath); err != nil {
			errs = append(errs, fmt.Errorf("write memprofile: %w", err))
		} else {
			log.Printf("editor heap profile written: %s", p.memProfilePath)
		}
	}

	if p.stopCPUProfile {
		runtimepprof.StopCPUProfile()
		p.stopCPUProfile = false
	}
	if p.cpuProfileFile != nil {
		if err := p.cpuProfileFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close cpuprofile: %w", err))
		} else {
			log.Printf("editor cpu profile written: %s", p.cpuProfileFile.Name())
		}
		p.cpuProfileFile = nil
	}

	if p.stopTrace {
		runtimetrace.Stop()
		p.stopTrace = false
	}
	if p.traceFile != nil {
		if err := p.traceFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close trace: %w", err))
		} else {
			log.Printf("editor execution trace written: %s", p.traceFile.Name())
		}
		p.traceFile = nil
	}

	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown pprof server: %w", err))
		}
		p.server = nil
	}

	if p.memProfileRateReset != nil {
		p.memProfileRateReset()
		p.memProfileRateReset = nil
	}

	return errors.Join(errs...)
}

func (p *profiler) captureMemProfiles(interval time.Duration) {
	for {
		select {
		case <-p.memTicker.C:
			p.memSnapshotIndex++
			snapshotPath := periodicHeapProfilePath(p.memProfilePath, p.memSnapshotIndex)
			if err := writeHeapProfile(snapshotPath); err != nil {
				log.Printf("write periodic heap profile after %s: %v", interval, err)
				continue
			}
			log.Printf("editor periodic heap profile written: %s", snapshotPath)
		case <-p.memTickerDone:
			return
		}
	}
}

func writeHeapProfile(path string) error {
	profileFile, err := createProfileFile(path)
	if err != nil {
		return err
	}
	defer profileFile.Close()

	runtime.GC()
	if err := runtimepprof.WriteHeapProfile(profileFile); err != nil {
		return err
	}
	return nil
}

func createProfileFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}

func periodicHeapProfilePath(basePath string, index int) string {
	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext)
	if ext == "" {
		ext = ".pprof"
	}
	return fmt.Sprintf("%s.%03d%s", name, index, ext)
}
