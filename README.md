# Defective

A lil' metroidvania starring a lil' robot. Built with Go + Ebitengine.

## Running the game 
You can run **Defective** with the following command:
```
go run . -level long_fall.json
```
Or build to the target OS. All assets are embedded in the executable.

## Profiling the editor
Run the editor with `pprof` enabled when you need to track heap growth:
```
go run ./cmd/editor -level long_fall.json -pprof localhost:6060
```

Then inspect the live heap profile:
```
go tool pprof http://localhost:6060/debug/pprof/heap
```

To capture a live CPU profile for 30 seconds instead:
```
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

Or write CPU samples to disk for the full editor session:
```
go run ./cmd/editor -level long_fall.json -cpuprofile tmp/editor-cpu.pprof
```

To write profiles to disk on exit, add `-cpuprofile cpu.pprof` and or `-memprofile heap.pprof`. If you want rolling heap snapshots while reproducing the leak, combine `-memprofile heap.pprof` with `-memprofile-sample 30s`.




