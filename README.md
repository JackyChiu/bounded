# Bounded Goroutine Management
`bounded` is a thin concurrency wrapper that provides bounded goroutine management.

Go is designed with lightweight concurrency primitives. Goroutines model concurrent tasks that could be executed by calling `go`.
Much programs end up having boundless concurrency and as a result they comes with producing a significant amount of overhead.

`bounded.Pool` is a bounded goroutine manager. It ensures that goroutines spawned are
within the given limit. The benefit being the ability to think and write go
programs without worrying about the overhead of spawning too much goroutines.

```go
pool, ctx := bounded.NewPool(context.Background(), 20)
...
for item := range itemStream {
  pool.Go(func() error {
    return processItem(ctx, item)
  })
}
```

Pool provides some simple synchronization and error capturing abilities too.
Developers can wait for all goroutines in the pool to complete and exit with
Wait(). The first error captured is returned.

```go
if err := pool.Wait(); err != nil {
  // handle error
}
```

Pool lazily spawns workers in the pool as tasks are queued up. Tasks are
favored to be completed by an existing worker. If all workers are busy then
it will spawn a new worker and enqueue the task again. This behaviour is
ongoing until the size of the pool has reached it's limit.

```go
pool, ctx := bounded.NewPool(context.Background(), HUGE_NUMBER)
...
for item := range itemStream {
  pool.Go(func() error {
    return fastProcess(ctx, item)
  })
}

// Pool will only use as much goroutines as the tasks need, up to the limit.
```
