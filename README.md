# `bounded` [![](https://circleci.com/gh/JackyChiu/bounded.svg?style=svg)](https://circleci.com/gh/JackyChiu/bounded)

## Bounded Goroutine Management
`bounded.Pool` is a bounded goroutine manager. Pool provides:
- Ensures goroutines spawned to be within the limit
- Lazily spawns new goroutines when ones in the pool are busy, up to the limit
- Captures the first non-nil `error` in the pool
- Notifies other goroutines through the cancellation the returned context
- Ability to wait for all goroutines to complete


```bash
go get github.com/JackyChiu/bounded
```

## Example
```go
pool, ctx := bounded.NewPool(context.Background(), 20)

for {
  select {
    case message := <-stream:
      pool.Go(func () error {
        // process message
      })
    case <-ctx.Done():
      // continue and check pool for error
  }
}

if err := pool.Wait(); err != nil {
  // handle error
}
```

## Why
Go encourages programming with concurrent design, allowing us to execute independent tasks with goroutines.
Much programs end up having boundless concurrency and as a result they comes with producing a significant amount of overhead.

This package is a attempt at providing an thin API (along with synchronization/error capturing built-in) to allow developers to continue programming with the same mental model without concern for the overhead.

## Synchronization And Error Capture
Pool provides some simple synchronization and error capturing abilities.
Developers can wait for all goroutines in the pool to complete and exit with
`Wait()`. If an error occurs in the pool, it's capture and the goroutines are
notified via `context.Context.Done()`. The first error captured is returned.

## Lazy
Pool lazily spawns workers in the pool as tasks are queued up. Tasks are
favored to be completed by an existing worker. If all workers are busy then
it will spawn a new worker and enqueue the task again. This behaviour is
ongoing until the size of the pool has reached it's limit.

```go
pool, ctx := bounded.NewPool(context.Background(), HUGE_NUMBER)

for message := range stream {
  pool.Go(func() error {
    // process message
  })
}

// Pool will reuse goroutines. New goroutines are spawned when the workers
// are busy, up to the pool limit.
```

## Gotchas

### Deadlocks

### Producer Channels
