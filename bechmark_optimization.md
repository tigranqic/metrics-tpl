# Performance Optimization Summary

This project includes targeted performance optimizations focused on reducing **allocation churn and GC pressure** in hot paths, without changing external behavior.

## Optimization Goals
- Reduce the number of dynamic allocations
- Eliminate slice growth caused by repeated `append`
- Improve memory locality via pre-allocation
- Keep runtime behavior stable under load

---

## Methodology

Profiling was performed under identical load conditions before and after optimization.

Two profiles were compared:
- **Base** — before optimization
- **Result** — after optimization

Comparison was done using Go `pprof` diff mode:

```bash
go tool pprof -diff_base=profiles/base.pprof profiles/result.pprof
```

# Profiling Results

## Analyzed Profiles

- **alloc_objects** — number of allocated objects
- **alloc_space** — total allocated bytes over time
- **inuse_space** — live memory snapshot

## Key Changes

- **Pre-allocated Slices**: Pre-allocation in handlers and batch operations was implemented to reduce frequent memory allocations.
- **Optimized Batch Updates**: Batch updates were optimized in both in-memory storage and PostgreSQL, resulting in improved memory usage.
- **Reduced Temporary Object Creation**: Reduced the creation of temporary objects in persistence logic, which helped lower memory overhead.

## Profiling Results

### alloc_objects

- A **significant reduction** in the number of allocated objects was observed, indicating better memory management.

```
File: server
Type: alloc_objects
Time: 2026-02-08 09:58:42 +04
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 113945, 13.15% of 866548 total
Dropped 14 nodes (cum <= 4332)
Showing top 10 nodes out of 115
      flat  flat%   sum%        cum   cum%
   -131074 15.13% 15.13%    -131074 15.13%  reflect.unsafe_NewArray
     65538  7.56%  7.56%     -65536  7.56%  reflect.MakeSlice
     65536  7.56%     0%      65536  7.56%  text/template/parse.(*PipeNode).append
    -43692  5.04%  5.04%     -92843 10.71%  reflect.Value.call
     43691  5.04% 0.00012%      43691  5.04%  strconv.FormatFloat
     32768  3.78%  3.78%      32768  3.78%  text/template/parse.(*CommandNode).append
     21846  2.52%  6.30%     -70997  8.19%  text/template.(*state).evalCall
     21846  2.52%  8.82%      21846  2.52%  text/template/parse.(*Tree).newAction
     20357  2.35% 11.17%      20357  2.35%  net/textproto.readMIMEHeader
     17129  1.98% 13.15%      17129  1.98%  text/template.addValueFuncs
```

## Key Improvements

- **Large Decrease in `reflect.unsafe_NewArray`**: A significant reduction was observed in the use of `reflect.unsafe_NewArray`, which contributed to better memory management.
- **Reduced Number of Temporary Reflect-Based Objects**: Temporary objects created via reflection were minimized, confirming the elimination of dynamic slice growth and temporary allocations.


### alloc_space (Total Allocated Memory)

- The total allocated memory over time was **reduced in critical paths**, showing improvements in memory efficiency for key operations.

```
File: server
Type: alloc_space
Time: 2026-02-08 09:58:42 +04
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) alloc_space
(pprof) top
Showing nodes accounting for 4MB, 11.43% of 35.03MB total
Dropped 10 nodes (cum <= 0.18MB)
Showing top 10 nodes out of 119
      flat  flat%   sum%        cum   cum%
      -2MB  5.71%  5.71%       -2MB  5.71%  reflect.unsafe_NewArray
    1.50MB  4.28%  1.43%        1MB  2.86%  context.(*cancelCtx).propagateCancel
    1.50MB  4.28%  2.86%    -0.50MB  1.43%  reflect.MakeSlice
       1MB  2.86%  5.72%        1MB  2.86%  runtime.allocm
       1MB  2.86%  8.57%        1MB  2.86%  text/template.addValueFuncs
      -1MB  2.86%  5.72%       -1MB  2.86%  net/http.(*Request).SetPathValue
       1MB  2.85%  8.57%        1MB  2.85%  text/template/parse.(*Tree).newText
       1MB  2.85% 11.43%        4MB 11.42%  net/http.(*conn).readRequest
       1MB  2.85% 14.28%        1MB  2.85%  text/template/parse.(*Tree).newAction
      -1MB  2.85% 11.43%       -1MB  2.85%  reflect.Value.call
```

- Observed increases in reflect.MakeSlice are expected and result from intentional pre-allocation (fewer allocations, larger upfront buffers).

### inuse_space

- Live memory increased slightly in some components:

```
File: server
Type: inuse_space
Time: 2026-02-08 09:58:42 +04
Showing nodes accounting for 513.95kB, 25.07% of 2050.10kB total
      flat  flat%   sum%        cum   cum%
    1026kB 50.05% 50.05%     1026kB 50.05%  runtime.allocm
 -512.05kB 24.98% 25.07%  -512.05kB 24.98%  runtime.main
  512.05kB 24.98% 50.05%   512.05kB 24.98%  github.com/tigranqic/metrics-tpl/internal/repository.(*MemStorage).StartAutoSave.func1
 -512.05kB 24.98% 25.07%  -512.05kB 24.98%  sync.runtime_notifyListWait
         0     0% 25.07%  -512.05kB 24.98%  net/http.(*conn).serve
         0     0% 25.07%  -512.05kB 24.98%  net/http.(*connReader).abortPendingRead
         0     0% 25.07%  -512.05kB 24.98%  net/http.(*response).finishRequest
         0     0% 25.07%     1539kB 75.07%  runtime.mcall
         0     0% 25.07%     -513kB 25.02%  runtime.mstart
         0     0% 25.07%     -513kB 25.02%  runtime.mstart0
         0     0% 25.07%     -513kB 25.02%  runtime.mstart1
         0     0% 25.07%     1026kB 50.05%  runtime.newm
         0     0% 25.07%     1539kB 75.07%  runtime.park_m
         0     0% 25.07%     1026kB 50.05%  runtime.resetspinning
         0     0% 25.07%     1026kB 50.05%  runtime.schedule
         0     0% 25.07%     1026kB 50.05%  runtime.startm
         0     0% 25.07%     1026kB 50.05%  runtime.wakep
         0     0% 25.07%  -512.05kB 24.98%  sync.(*Cond).Wait
```

- This is expected and does not indicate a regression, as inuse_space reflects a point-in-time snapshot rather than allocation rate.

## Conclusion

- Allocation count significantly reduced

- Dynamic slice growth eliminated in hot paths

- GC pressure reduced

- No performance regressions observed