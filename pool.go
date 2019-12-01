package bounded

import (
	"context"
	"sync"
	"sync/atomic"
)

// Pool is a bounded goroutine manager. It ensures that goroutines spawned are
// within the given limit. The benefit being the ability to think and write go
// programs without worrying about the overhead of spawning too much goroutines.
//
// Pool provides some simple synchronization and error capturing abilities too.
// Developers can wait for all goroutines in the pool to complete and exit with
// Wait(). The first error captured is returned.
//
// Pool lazily spawns workers in the pool as tasks are queued up. Tasks are
// favored to be completed by an existing worker. If all workers are busy then
// it will spawn a new worker and enqueue the task again. This behaviour is
// ongoing until the size of the pool has reached it's limit.
type Pool struct {
	errPool errorPool
	ctx     context.Context

	tasks     chan func() error
	closeOnce sync.Once
	// taskWg is used for task completion synchronization in the pool.
	taskWg sync.WaitGroup

	limit uint32
	size  uint32
}

// NewPool returns a Pool instances and a new context. The number of goroutines
// spawned are limited by the given max capacity. The new context includes
// cancellations from the goroutines in the Pool.
func NewPool(ctx context.Context, poolSize uint32) (*Pool, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		errPool: errorPool{
			cancel: cancel,
		},
		ctx:   ctx,
		tasks: make(chan func() error),
		limit: poolSize,
	}
	p.startWorker()
	return p, ctx
}

// Go will enqueue the task for execution by one of goroutines in the pool.
// Calls to Go will spin up workers lazily, as the workers are blocked, new
// workers will be spawned until the goroutine limit has been reached.
func (p *Pool) Go(task func() error) {
	p.taskWg.Add(1)

	if p.Size() < p.limit {
		// This code path is only used while the Pool is still lazily
		// loading goroutines.
		select {
		case p.tasks <- task:
			return
		case <-p.ctx.Done():
			p.taskWg.Done()
			return
		default:
		}
		// Failed sends to the task channel tell us that the workers are busy.
		// Start a new worker to help execute tasks.
		p.startWorker()
	}

	select {
	case p.tasks <- task:
	case <-p.ctx.Done():
		p.taskWg.Done()
	}
}

// Wait waits for the tasks and worker goroutines in the pool to exit.
// The first error to occur in the pool is returned, if any.
func (p *Pool) Wait() error {
	// Waits for the tasks to finish execution. When this is confirmed,
	// the task channels can be closed.
	p.taskWg.Wait()

	p.closeOnce.Do(func() {
		close(p.tasks)
	})

	// Finally we wait for the worker goroutines to exit.
	return p.errPool.Wait()
}

// Size is the number of goroutines running in the pool.
func (p *Pool) Size() uint32 {
	return atomic.LoadUint32(&p.size)
}

func (p *Pool) incrementSize() {
	atomic.AddUint32(&p.size, 1)
}

// startWorker spins up a worker ready to execute incoming tasks.
func (p *Pool) startWorker() {
	p.errPool.Go(func() error {
		for {
			select {
			case task, ok := <-p.tasks:
				if !ok {
					return nil
				}
				p.errPool.execute(task)
				p.taskWg.Done()
			case <-p.ctx.Done():
				return p.ctx.Err()
			}
		}
	})
	p.incrementSize()
}

// errPool manages a group of goroutines and allows for error tracking.
type errorPool struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup

	errOnce sync.Once
	err     error
}

// Go spins up a goroutine to execute the task.
func (e *errorPool) Go(task func() error) {
	e.wg.Add(1)

	go func() {
		e.execute(task)
		e.wg.Done()
	}()
}

// Wait waits for all goroutines to exit and returns the first error that
// occurred, if any.
func (e *errorPool) Wait() error {
	e.wg.Wait()
	if e.cancel != nil {
		e.cancel()
	}
	return e.err
}

// execute runs the task and records the first error that occurs.
// This in turn cancels any other tasks.
func (e *errorPool) execute(task func() error) {
	err := task()
	if err == nil {
		return
	}
	e.errOnce.Do(func() {
		e.err = err
		if e.cancel != nil {
			e.cancel()
		}
	})
}
