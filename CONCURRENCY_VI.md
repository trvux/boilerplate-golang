# Hướng Dẫn Sử Dụng Concurrency, Parallelism Và Asynchronous Trong Go

[English README](README.md) | [Tiếng Việt](README_VI.md) | [Concurrency Guide (EN)](CONCURRENCY.md) | [Hướng Dẫn Concurrency (VI)](CONCURRENCY_VI.md)

Ngôn ngữ Go nổi tiếng thế giới nhờ mô hình xử lý đồng thời cực kỳ mạnh mẽ và tối ưu. Tài liệu này hướng dẫn chi tiết các tiêu chuẩn, kiến trúc và mô hình thiết kế (design patterns) để xử lý tác vụ đồng thời (concurrency), chạy song song (parallelism) và bất đồng bộ (async) một cách an toàn, hiệu quả trong Boilerplate này.


---

## 1. Concurrency (Đồng Thời) vs Parallelism (Song Song) Trong Go

Hiểu rõ sự khác biệt giữa hai khái niệm này là bắt buộc đối với một nhà phát triển cấp Senior/Tech Lead:

- **Concurrency (Đồng thời - Cấu trúc)**: Cách thiết kế cấu trúc chương trình thành các tiến trình độc lập hoạt động đan xen nhau (các goroutines). Đây là câu chuyện về *cấu trúc thiết kế*. Go quản lý đồng thời một cách tự nhiên thông qua Goroutines và Channels, được điều phối bởi Go Runtime Scheduler.
- **Parallelism (Song song - Thực thi)**: Cách thực thi nhiều tính toán cùng một thời điểm vật lý trên các vi xử lý đa nhân (multi-core CPUs). Đây là câu chuyện về *thực thi*. Go đạt được tính song song bằng cách tự động ánh xạ hàng ngàn goroutines đồng thời lên các luồng (threads) của hệ điều hành một cách linh hoạt (được điều khiển bởi cài đặt `runtime.GOMAXPROCS`, mặc định bằng số nhân CPU vật lý).

---

## 2. Mô Hình Bất Đồng Bộ An Toàn (Asynchronous / Fire-and-Forget)

Khi cần kích hoạt các tác vụ chạy ngầm (như gửi email thông báo, đẩy tin nhắn sự kiện vào Kafka, hoặc nạp đệm cache) mà không muốn làm nghẽn luồng xử lý HTTP chính của khách hàng, chúng ta thực hiện bất đồng bộ.

### Cái Bẫy Chết Người (Fatal Panic) Trong Goroutines
Trong Go, nếu một goroutine được tách ra gặp sự cố nghiêm trọng (runtime panic - như lỗi gọi con trỏ nil) mà **không được khôi phục (recover) kịp thời, toàn bộ ứng dụng Go sẽ sập lập tức**. Middleware Recovery của Gin hoàn toàn không thể bắt được các panic phát sinh bên trong các goroutines chạy ngầm độc lập này.

### Giải Pháp: Bộ Bọc Khởi Chạy Goroutine An Toàn
Luôn luôn bọc các tác vụ chạy ngầm trong một hàm có defer recover. Bộ bọc an toàn dưới đây đã được tích hợp làm quy chuẩn trong dự án:

```go
package async

import (
	"runtime/debug"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

// GoSafe khởi chạy một goroutine an toàn, tự động bắt và xử lý mọi panic runtime
// để tránh làm sập toàn bộ ứng dụng chính.
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

### Cách áp dụng thực tế trong tầng Usecase:
```go
func (u *productUsecase) AddStock(ctx context.Context, id uint64, qty int) (*domain.Product, error) {
	p, err := u.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Thực hiện cập nhật cơ sở dữ liệu đồng bộ (Blocking/Synchronous)
	if err := p.AddStock(qty); err != nil {
		return nil, err
	}
	if err := u.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// 2. Kích hoạt tác vụ gửi event bất đồng bộ chạy ngầm an toàn (Non-blocking Async)
	async.GoSafe(u.log, func() {
		// Bắt buộc dùng Context mới độc lập (context.Background) 
		// vì HTTP Request Context (ctx) sẽ bị hủy ngay khi API trả về kết quả cho client!
		bgCtx := context.Background()
		_ = u.eventProducer.Publish(bgCtx, "stock-adjusted", []byte(p.SKU), []byte("stock added"))
	})

	return p, nil
}
```

---

## 3. Chạy Tác Vụ Song Song Đồng Thời (Fan-Out / Fan-In)

Khi ứng dụng của bạn cần lấy dữ liệu đồng thời từ nhiều nguồn độc lập (ví dụ: gọi đồng thời hai API dịch vụ bên ngoài, hoặc chạy song song hai câu lệnh SQL độc lập) và đợi tất cả hoàn thành để gộp dữ liệu, hãy sử dụng thư viện `golang.org/x/sync/errgroup`.

`errgroup.Group` vượt trội hơn `sync.WaitGroup` truyền thống nhờ:
1. Tự động lan truyền lỗi đầu tiên gặp phải của các goroutines con về hàm gọi chính.
2. Hỗ trợ hủy bỏ (context cancellation) tự động các tác vụ song song khác đang chạy nếu một tác vụ bất kỳ bị lỗi.

### Ví dụ triển khai Fan-Out song song:
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

	// Tác vụ 1: Lấy danh sách sản phẩm song song
	g.Go(func() error {
		var err error
		products, err = u.repo.List(gCtx, 0, 100)
		return err // nếu DB lỗi, lập tức báo về group
	})

	// Tác vụ 2: Tính toán giá trị kho song song
	g.Go(func() error {
		var err error
		totalValue, err = u.repo.CalculateTotalStockValue(gCtx)
		return err
	})

	// Hàm Wait sẽ chặn lại cho đến khi mọi tác vụ chạy xong 
	// HOẶC có ít nhất một tác vụ trả về lỗi phi-nil.
	if err := g.Wait(); err != nil {
		return nil, err // trả lỗi ra ngoài một cách tường minh
	}

	return &DashboardData{
		Products:   products,
		TotalValue: totalValue,
	}, nil
}
```

---

## 4. Kiểm Soát Giới Hạn Tài Nguyên (Worker Pool Pattern)

Để tránh cạn kiệt tài nguyên hệ thống (như mở quá nhiều kết nối cơ sở dữ liệu đồng thời, hoặc gọi quá tải API của bên thứ ba), bạn phải kiểm soát số lượng goroutines chạy đồng thời tối đa thông qua cấu trúc **Worker Pool** kết hợp với Go Channels.

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

	// Tạo số lượng workers giới hạn cố định
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
						return // channel đã đóng, không còn việc để làm
					}
					// Thực thi tác vụ xử lý công việc
					err := processJob(job)
					results <- Result{JobID: job.ID, Error: err}
				}
			}
		}(w)
	}

	// Đóng channel kết quả khi tất cả các workers đã hoàn thành
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

## 5. Truyền Dẫn Và Hủy Bỏ Qua Context (Context Lifecycle)

Quản lý vòng đời thông qua `context.Context` là quy chuẩn bắt buộc trong các hệ thống Go chuyên nghiệp:

1. **Lan truyền sự hủy bỏ (Cancellation Propagation)**: Khi người dùng ngắt kết nối (ví dụ tắt trình duyệt giữa chừng) hoặc yêu cầu bị hết thời gian chờ (Timeout), HTTP Request Context (`c.Request.Context()`) của Gin sẽ tự động bị hủy. Nếu bạn truyền dẫn biến `ctx` này xuyên suốt từ Handler qua Usecase xuống GORM Repository, GORM sẽ tự động hủy ngay câu lệnh SQL đang thực thi trên PostgreSQL, giải phóng tài nguyên DB lập tức.
2. **Theo vết ngữ cảnh (Context Metadata)**: GORM và Zap Logger nạp thông tin Request ID từ `context.Context` để xâu chuỗi toàn bộ hoạt động của một yêu cầu.

### Quy tắc truyền Context đúng cách
Luôn khai báo tham số `ctx context.Context` làm tham số đầu tiên trong mọi chữ ký hàm của Service/Usecase và Repository:
```go
// Chữ ký hàm chuẩn SOLID
GetByID(ctx context.Context, id uint64) (*domain.Product, error)
```

### Các nguyên tắc an toàn:
- **Nguyên tắc 1**: KHÔNG BAO GIỜ truyền một Context đã bị hủy hoặc sắp bị hủy vào các tác vụ chạy ngầm bất đồng bộ. Đối với các goroutine chạy ngầm, hãy khởi tạo một Context hoàn toàn mới thông qua `context.Background()`.
- **Nguyên tắc 2**: Không bao giờ lưu trữ biến `Context` làm thuộc tính bên trong một Struct. Context chỉ được truyền trực tiếp làm tham số của hàm.

---

## 6. Lỗi Xung Đột Bộ Nhớ (Race Conditions) Và Chia Sẻ Trạng Thái

Trong thiết kế Modular Monolith Clean Architecture của chúng ta:
- Các cấu trúc Handlers, Usecases và Repositories được khởi tạo một lần duy nhất lúc khởi động ứng dụng (mô hình **Stateless Singletons**).
- Vì nhiều yêu cầu HTTP đồng thời sẽ chạy qua các Singleton này cùng một lúc, **KHÔNG BAO GIỜ được khai báo các thuộc tính chứa trạng thái của riêng request bên trong Struct**.

### Phòng Chống Xung Đột Dữ Liệu (Data Race)
Trong trường hợp bắt buộc phải duy trì một trạng thái có thể thay đổi được trong bộ nhớ dùng chung (như bộ nhớ đệm cache local, bộ đếm giới hạn tần suất truy cập rate limiter), bạn bắt buộc phải bảo vệ dữ liệu đó bằng Read-Write Mutex (`sync.RWMutex`) hoặc các hoạt động nguyên tử (`sync/atomic`):

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

### Kiểm tra Data Race lúc Dev/CI
Luôn luôn xác minh an toàn đồng thời trong quá trình code và chạy CI bằng cách kích hoạt cờ kiểm tra race detector:
```bash
go test -race ./...
```
Nếu phát hiện bất kỳ xung đột tài nguyên dùng chung nào, Go sẽ in chi tiết stack trace chỉ rõ vị trí dòng code đọc và ghi dữ liệu đồng thời đang bị xung đột.
