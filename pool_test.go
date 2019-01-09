package bounded

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	poolSize := 5
	pool, _ := NewPool(context.Background(), poolSize)

	results := make(chan int)

	for i := 0; i < poolSize; i++ {
		i := i
		pool.Go(func() error {
			results <- i
			return nil
		})
	}

	for i := 0; i < poolSize; i++ {
		<-results
	}

	if err := pool.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPool_limits_goroutines(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolSize := 50
	pool, ctx := NewPool(ctx, poolSize)

	for i := 0; i < 100; i++ {
		pool.Go(func() error {
			// "heavy" task
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	if err := pool.Wait(); err != nil {
		t.Errorf("expected no errors, got: %v", err)
	}

	if size := pool.Size(); size > poolSize {
		t.Errorf("expected goroutines to cap at %v, got: %v", poolSize, size)
	}
}

func TestPool_lazily_loads_goroutines(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolSize := 50
	pool, ctx := NewPool(ctx, poolSize)

	for i := 0; i < 50; i++ {
		pool.Go(func() error {
			// "light" task
			return nil
		})
		time.Sleep(5 * time.Millisecond)
	}

	if err := pool.Wait(); err != nil {
		t.Errorf("expected no errors, got: %v", err)
	}

	if size := pool.Size(); size >= poolSize {
		t.Errorf("expected goroutines to run lazily, got: %v", size)
	}
}

func TestPool_exits_when_context_is_cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, ctx := NewPool(ctx, 5)
	var events int32

	for i := 0; i < 100; i++ {
		if i >= 80 {
			cancel()
		}
		pool.Go(func() error {
			atomic.AddInt32(&events, 1)
			return nil
		})
	}

	if err := pool.Wait(); err != context.Canceled {
		t.Errorf("expected error to be context.Canceled, got: %v", err)
	}

	if events >= 100 {
		t.Errorf("expected goroutines to exit early, got: %v", events)
	}
}

func TestPool_exits_on_error(t *testing.T) {
	expectedErr := errors.New("error")
	pool, _ := NewPool(context.Background(), 5)
	var events int32

	for _, err := range []error{
		expectedErr,
		errors.New("pool_test: 1"),
		errors.New("pool_test: 2"),
	} {
		err := err
		pool.Go(func() error {
			atomic.AddInt32(&events, 1)
			return err
		})
	}

	if err := pool.Wait(); expectedErr != err {
		t.Errorf("expected error to be %v, got: %v", expectedErr, err)
	}

	if events != 1 {
		t.Errorf("expected only 1 goroutine to run and error, got: %v", events)
	}
}

func TestZeroGroup(t *testing.T) {
	err1 := errors.New("pool_test: 1")
	err2 := errors.New("pool_test: 2")

	cases := []struct {
		errs []error
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err2}},
	}

	for _, tc := range cases {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var firstErr error
		pool, _ := NewPool(ctx, 1)

		for _, err := range tc.errs {
			err := err
			pool.Go(func() error { return err })
			if firstErr == nil && err != nil {
				firstErr = err
			}
		}
		if actualErr := pool.Wait(); actualErr != firstErr {
			t.Errorf("unexpected error, expected: %v, got: %v", firstErr, actualErr)
		}
	}
}

func TestWithContext(t *testing.T) {
	errDoom := errors.New("group_test: doomed")

	cases := []struct {
		errs []error
		want error
	}{
		{want: nil},
		{errs: []error{nil}, want: nil},
		{errs: []error{errDoom}, want: errDoom},
		{errs: []error{errDoom, nil}, want: errDoom},
	}

	for _, tc := range cases {
		ctx := context.Background()
		pool, ctx := NewPool(ctx, 5)

		for _, err := range tc.errs {
			err := err
			pool.Go(func() error { return err })
		}

		if err := pool.Wait(); err != tc.want {
			t.Errorf("unexpected error, expected: %v, got: %v", tc.want, err)
		}

		canceled := false
		select {
		case <-ctx.Done():
			canceled = true
		default:
		}
		if !canceled {
			t.Error("expected context to be canceled")
		}
	}
}
