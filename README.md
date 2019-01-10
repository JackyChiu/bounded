# `bounded` [![](https://circleci.com/gh/JackyChiu/bounded.svg?style=svg)](https://circleci.com/gh/JackyChiu/bounded) [![Documentation](https://godoc.org/github.com/JackyChiu/bounded?status.svg)](https://godoc.org/github.com/JackyChiu/bounded/)

## Bounded Goroutine Management
`bounded.Pool` is a bounded goroutine manager. Pool provides:
- Ensures goroutines spawned to be within the limit
- Lazily spawns new goroutines when ones in the pool are busy, up to the limit
- Captures the first non-nil `error` in the pool
- Notifies other goroutines through the cancellation the returned context
- Ability to wait for all goroutines to complete
- Insignificant to no overhead compared to hand rolled worker pools

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

For a more complete example checkout the
[examples directory](https://github.com/JackyChiu/bounded/blob/master/examples/bounded/md5_all.go).

## Why
Go encourages programming with concurrent design, allowing us to execute
independent tasks with goroutines.  Much programs end up having boundless
concurrency and as a result they comes with producing a significant amount of
overhead.

This package is a attempt at providing an thin API (along with synchronization/
error capturing built-in) to allow developers to continue programming with the
same mental model without concern for the overhead.

## Synchronization And Error Capture
Pool provides simple synchronization and error capturing abilities. It provides
an API to wait for all tasks in the pool to complete and exit with `Wait()`. If
an error occurs in the pool, it's capture and the goroutines are notified via
`context.Context.Done()`. The main goroutine can also use this to tell if it
should stop producing work to the pool. The first error captured is returned.

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
Calls to `Pool.Go` will block when a worker isn't available. This is a issue
when you're designing a producer/consumer type system where you want to spawned
workers to produce results and consume the results them in the same goroutine.

```go
results := make(chan results)
for message := range stream {
  // This can block and cause a deadlock becauase nothing is able to consume
  // from the results channel.
  pool.Go(func() error {
    // process message to sent to results channel
  })
}
go func() {
  pool.Wait()
  close(results)
}

for r := range results {
  // process results
}
```

Instead consider have a separate goroutine to consume the messages and spawn
the workers from there. This will also allow you to have a goroutine
responsible for closing the `results` channel.

```go
results := make(chan results)
go func() {
  defer close(results)
  // This goroutines
  for message := range stream {
    pool.Go(func() error {
      // process message to send to results channel
    })
  }
  pool.Wait()
}()

for r := range results {
  // process results
}

if err := pool.Wait(); err != nil {
  // handle error
}
```

### Performance

In the producer/consumer model shown above, there is only one goroutine
consuming messages from the `messages` channel. Compared to possible hand roll
solutions that have multiple goroutines reading from the channel. This applies
back pressure to the producer at the cost of the producer being blocked.
To increase performance of the producer give the `messages` channel a buffer so
that it isn't as blocked as often.

```go
pool, ctx := bounded.NewPool(context.Background(), 20)

messages := make(chan message, 10)
pool.Go(func() error {
  defer close(messages)
  return produceMessages(messages)
})

// Consume messages

```
