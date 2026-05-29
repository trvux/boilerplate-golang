# Concurrency, Parallelism, and Asynchronous Patterns Guide

[English README](README.md) | [Tiếng Việt](README_VI.md) | [Concurrency Guide (EN)](CONCURRENCY.md) | [Hướng Dẫn Concurrency (VI)](CONCURRENCY_VI.md)

Go is world-renowned for its powerful concurrency model. This guide outlines standard practices, architectures, and design patterns for handling concurrency, parallelism, and asynchronous operations safely within this boilerplate.


---

## 1. Concurrency vs Parallelism in Go

Understanding the distinction is vital for writing high-performance Go applications:

- **Concurrency (Cấu trúc)**: Designing your program as a collection of independent, concurrent processes (goroutines). It is about *structure*. Go handles concurrency natively through goroutines and channels, managed by the Go runtime scheduler.
- **Parallelism (Thực thi)**: Executing multiple computations simultaneously on multi-core CPUs. It is about *execution*. Go achieves parallelism by mapping concurrent goroutines to multiple operating system threads dynamically (governed by the `runtime.GOMAXPROCS` setting, which defaults to the number of CPU cores).

---

## 2. Safe Asynchronous Patterns (Fire-and-Forget)

When triggering background jobs (like sending notification emails, publishing event messages, or warming caches) that should not block the main HTTP request flow, we execute them asynchronously.

### The Fatal Panic Trap in Goroutines
In Go, if a spawned goroutine encounters a runtime panic (like a nil pointer dereference) and does not recover, the **entire application will crash immediately**. A panic inside a background goroutine is not caught by the Gin middleware recovery.

### The Solution: Safe Goroutine Wrapper
Always wrap asynchronous goroutines with a recovery block. Below is the standard pattern implemented in this boilerplate:

```go
package async

import (
	"runtime/debug"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

// GoSafe starts a goroutine safely, intercepting any runtime panics
// and preventing the entire application from crashing.
func GoSafe(log logger.Logger, task func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error("Background goroutine panic intercepted",
					zap.Any("panic_err", r),
					zap.String("stack", string(stack)),
				)
			}
		}()
		task()
	}()
}
```

### Usage in Usecases:
```go
func (u *productUsecase) AddStock(ctx context.Context, id uint64, qty int) (*domain.Product, error) {
	p, err := u.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Perform database updates synchronously
	if err := p.AddStock(qty); err != nil {
		return nil, err
	}
	if err := u.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// 2. Trigger non-blocking async events (e.g., publish transaction logs to Kafka) safely
	async.GoSafe(u.log, func() {
		// Use Background context because the HTTP Request context (ctx) 
		// will be cancelled as soon as the HTTP handler returns!
		bgCtx := context.Background()
		_ = u.eventProducer.Publish(bgCtx, "stock-adjusted", []byte(p.SKU), []byte("stock added"))
	})

	return p, nil
}
```

---

## 3. Parallel Task Executions (Fan-Out / Fan-In)

When your application needs to fetch data from multiple independent sources simultaneously (e.g., calling two external APIs or running two separate SQL queries in parallel) and wait for all of them to finish, use `golang.org/x/sync/errgroup`.

`errgroup.Group` is superior to a standard `sync.WaitGroup` because:
1. It automatically propagates the first encountered error back to the caller.
2. It supports context cancellation, automatically aborting all other active concurrent tasks if one task fails.

### Parallel Fan-Out Implementation
```go
package usecase

import (
	"context"
	"golang.org/x/sync/errgroup"
	"github.com/tranvux/boilerplate_golang/internal/modules/product/domain"
)

func (u *productUsecase) FetchDashboardData(ctx context.Context) (*DashboardData, error) {
	g, gCtx := errgroup.WithContext(ctx)

	var products []*domain.Product
	var totalValue float64

	// Task 1: Fetch product catalog list in parallel
	g.Go(func() error {
		var err error
		products, err = u.repo.List(gCtx, 0, 100)
		return err // returns error if database fails
	})

	// Task 2: Calculate sum calculations in parallel
	g.Go(func() error {
		var err error
		totalValue, err = u.repo.CalculateTotalStockValue(gCtx)
		return err
	})

	// Wait blocks until all tasks complete or one task returns a non-nil error
	if err := g.Wait(); err != nil {
		return nil, err // propagates error cleanly
	}

	return &DashboardData{
		Products:   products,
		TotalValue: totalValue,
	}, nil
}
```

---

## 4. Bounded Concurrency (Worker Pool Pattern)

To prevent resource exhaustion (e.g., running too many database connections or making too many concurrent API calls at once), you must restrict the number of active goroutines using a bounded **Worker Pool** powered by Go Channels.

```go
package worker

import (
	"context"
	"fmt"
	"sync"
)

type Job struct {
	ID   int
	Data string
}

type Result struct {
	JobID int
	Error error
}

func StartWorkerPool(ctx context.Context, numWorkers int, jobs <-chan Job, results chan<- Result) {
	var wg sync.WaitGroup

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return // channel closed, no more jobs
					}
					// Process the job
					err := processJob(job)
					results <- Result{JobID: job.ID, Error: err}
				}
			}
		}(w)
	}

	// Close results channel once all workers are finished
	go func() {
		wg.Wait()
		close(results)
	}()
}

func processJob(j Job) error {
	fmt.Printf("Processing job %d: %s\n", j.ID, j.Data)
	return nil
}
```

---

## 5. Context Tracing and Cancellations

Context propagation is the gold standard in production-grade Go architectures:

1. **Context Cancellation Propagation**: When an HTTP client disconnects (closes their browser) or a request times out, Gin's `c.Request.Context()` is cancelled. If you propagate this `ctx` down through the Usecase to GORM/Postgres, GORM immediately cancels the active database query, freeing database resources.
2. **Context Metadata (Tracing)**: GORM and Zap loggers can parse request metadata (like the Request ID) from `context.Context` to link concurrent executions together.

### Propagating context correctly
Always pass `ctx` as the very first argument of your service and repository signatures:
```go
// Correct SOLID interface
GetByID(ctx context.Context, id uint64) (*domain.Product, error)
```

### Safety Rules:
- **Rule 1**: NEVER pass an expired or cancelled context to an asynchronous background goroutine. If your background routine needs a context, create a clean one using `context.Background()` or `context.TODO()`.
- **Rule 2**: Never store Contexts inside a struct; always pass them explicitly as parameters to functions.

---

## 6. Race Conditions and Shared States

In Go's Clean Architecture modular monolith:
- Handlers and Usecases are designed as **Stateless Singletons** wired once on startup.
- Because multiple HTTP requests invoke Usecase methods concurrently, **NEVER store request-specific states inside struct fields**.

### Data Race Prevention
If you must maintain a mutable shared state in memory (e.g. an in-memory cache, rate limiter, or metric counter), protect it using a Read-Write Mutex (`sync.RWMutex`) or atomic operations (`sync/atomic`):

```go
package cache

import "sync"

type ThreadSafeCache struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewCache() *ThreadSafeCache {
	return &ThreadSafeCache{
		data: make(map[string]string),
	}
}

func (c *ThreadSafeCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *ThreadSafeCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}
```

### Race Detection
Always validate concurrent safety during development and CI by running your test suites with the race detector flag:
```bash
go test -race ./...
```
If any data race is detected, Go will generate a detailed stack trace showing the concurrent conflicting read and write instructions.
