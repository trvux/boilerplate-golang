# Boilerplate Golang Modular Monolith Clean Architecture

[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Docker%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)](https://docker.com)

[English README](README.md) | [Tiếng Việt](README_VI.md) | [Concurrency Guide (EN)](CONCURRENCY.md) | [Hướng Dẫn Concurrency (VI)](CONCURRENCY_VI.md)


Tài liệu này cung cấp hướng dẫn chi tiết về cấu trúc, thiết lập và vận hành một Boilerplate Golang cấp Senior/Tech Lead. Dự án sử dụng phiên bản Go 1.26, triển khai theo mô hình Modular Monolith kết hợp kiến trúc Clean Architecture và các nguyên lý SOLID một cách nghiêm ngặt, giúp dễ dàng chia tách thành các dịch vụ microservices trong tương lai với thay đổi tối thiểu ở tầng logic nghiệp vụ.


Các công nghệ và cơ chế cốt lõi được tích hợp:
- Web Framework: Gin
- Database ORM: GORM (PostgreSQL)
- Quản lý cơ sở dữ liệu: Goose (nhúng trực tiếp các file SQL chạy tự động khi khởi động)
- Lưu trữ Cache: Redis
- Truyền thông điệp (Event Messaging): Kafka (KRaft mode chạy container đơn giản không cần Zookeeper)
- Dependency Injection (DI): Khởi tạo thủ công (được trang bị sẵn cấu hình Google Wire tự động để dự phòng)
- Trình ghi log cấu trúc: Uber Zap
- Quản lý lỗi: Phân cấp lỗi Domain và dịch lỗi tập trung (Error Translator)
- Theo vết yêu cầu: Middleware Request ID tự động lan truyền
- Môi trường cấu hình: Quản lý qua biến môi trường hệ thống kết hợp nạp file .env tự động (qua godotenv)

---

## Luồng Hoạt Động Của Kiến Trúc

Kiến trúc Clean Architecture quy định các luồng phụ thuộc phải đi hướng vào trong, trong khi luồng xử lý thực tế di chuyển qua các lớp vật lý.

### Luồng Xử Lý Yêu Cầu (Request-Response Flow)
```
HTTP Request  -> Delivery (Gin Handler) -> Usecase (Logic) -> Repository (GORM) -> PostgreSQL DB
HTTP Response <- Delivery (Gin Handler) <- Usecase (Logic) <- Repository (GORM) <- PostgreSQL DB
```

### Luồng Phụ Thuộc Ngược (Strict Dependency Inversion)
Tầng ngoài phụ thuộc hoàn toàn vào các Interface trừu tượng được định nghĩa ở tầng Domain bên trong. Nhờ đó, các thay đổi về cơ sở dữ liệu hoặc web framework ở ngoài không làm ảnh hưởng tới core business logic bên trong.
```
[Tầng Delivery]        [Tầng Usecase]         [Tầng Domain]         [Tầng Repository]
ProductHandler  --->  ProductUsecase (I)  <--  productUsecase (Impl)
                             |
                             v
                      ProductRepository (I) <-- postgresProductRepo (Impl)
```
Chú thích: (I) đại diện cho Interface (Trừu tượng), và (Impl) đại diện cho cấu trúc triển khai thực tế (Implementation).

---

## Cấu Trúc Thư Mục Dự Án

```
.
├── cmd/
│   └── server/
│       └── main.go         # Điểm khởi chạy ứng dụng (nạp cấu hình, chạy migration, graceful shutdown)
├── database/
│   ├── migrations/         # Thư mục lưu trữ các file script SQL migration dạng Goose
│   └── migrations.go       # File nhúng SQL script vào mã nguồn biên dịch bằng go:embed
├── internal/
│   ├── app/
│   │   ├── app.go          # Container ứng dụng (nối dây các module thủ công, quản lý vòng đời server)
│   │   ├── middleware.go   # Bộ middlewares Gin (Zap log request, phục hồi panic, CORS, RequestID)
│   │   └── wire.go         # Bản thiết kế DI tự động Google Wire dự phòng
│   └── modules/
│       └── product/        # Module mẫu tự chứa (hoàn toàn cô lập, dễ tách thành microservice)
│           ├── delivery/   # Tầng vận chuyển (Gin HTTP Handlers, request/response DTOs)
│           ├── domain/     # Tầng nghiệp vụ cốt lõi (Entities, điều kiện ràng buộc và các Interfaces)
│           ├── repository/ # Tầng dữ liệu (Adapter GORM kết nối và truy vấn PostgreSQL)
│           └── usecase/    # Tầng xử lý luồng nghiệp vụ (Business logic)
├── pkg/
│   ├── apperr/             # Khai báo cấu trúc lỗi Domain tập trung
│   ├── config/             # Bộ nạp cấu hình trực tiếp từ biến môi trường với godotenv dự phòng
│   ├── database/           # Bộ kết nối PostgreSQL, Redis và các hàm kiểm tra kết nối (ping/health)
│   ├── logger/             # Trình ghi log Zap cấu trúc cao cấp
│   ├── messaging/          # Bộ bọc thư viện Kafka Producer/Consumer (segmentio)
│   └── response/           # Tiêu chuẩn hóa định dạng JSON phản hồi (thành công/lỗi)
├── Dockerfile              # Dockerfile đa tầng tối ưu hóa dung lượng và bảo mật (Alpine, non-root)
├── docker-compose.yml      # Dựng môi trường phát triển cục bộ (Postgres, Redis, Kafka, App)
├── go.mod                  # File quản lý thư viện dự án Go
└── .env.example            # File mẫu cấu hình biến môi trường chạy local cho nhà phát triển
```

---

## Quản Lý Cấu Hình Và Biến Môi Trường

Hệ thống nạp cấu hình linh hoạt theo thứ tự ưu tiên:
1. Biến môi trường hệ thống (OS Environment Variables) có quyền ưu tiên cao nhất.
2. File cấu hình `.env` cục bộ (chỉ sử dụng khi chạy local và không được commit lên Git).
3. Các cấu hình mặc định được gán trực tiếp (fallback defaults) trong mã nguồn Go (không cần file cấu hình ngoài).

Xem danh sách đầy đủ các biến cấu hình tại `.env.example`.

---

## Hướng Dẫn Thiết Lập Và Khởi Chạy Hệ Thống

### Điều Kiện Tiên Quyết
- Máy tính của bạn đã cài đặt Docker và Docker Compose.
- Phiên bản Go 1.26 trở lên (chỉ bắt buộc nếu chạy ứng dụng trực tiếp trên local ngoài container).

---

### Phương Án 1: Chạy Bằng Docker Compose (Khuyên Dùng)

Phương án này tự động tải về, cài đặt và liên kết toàn bộ cơ sở dữ liệu, dịch vụ cache, message broker và tự động build mã nguồn ứng dụng Go của bạn thành một container bảo mật.

Bước 1: Mở terminal tại thư mục gốc của dự án và chạy lệnh sau:
```bash
docker compose up --build
```

Bước 2: Hệ thống sẽ tự động khởi tạo theo quy trình sau:
1. Container PostgreSQL khởi động và tạo cơ sở dữ liệu `boilerplate_db`.
2. Container Redis khởi động và kiểm tra trạng thái hoạt động.
3. Container Kafka khởi động ở chế độ KRaft độc lập trên cổng 9092.
4. Container ứng dụng Go khởi chạy sau khi kiểm tra các dịch vụ trên đã hoàn toàn sẵn sàng (healthy).
5. Ứng dụng Go tự động thực thi các file Goose SQL migration đã được nhúng sẵn trong binary để tạo bảng `products`.
6. Máy chủ Gin API khởi động thành công và lắng nghe tại địa chỉ `http://localhost:8080`.

Kiểm tra cổng dịch vụ chạy local sau khi Docker Compose Up thành công:
- Gin API Server: `http://localhost:8080` (Cổng dịch vụ chính)
- PostgreSQL Database: `localhost:5432` (Username: `postgres`, Password: `password`)
- Redis Cache: `localhost:6379`
- Kafka Brokers: `localhost:9092`

Để tắt toàn bộ hệ thống và giải phóng dữ liệu bộ nhớ đệm:
```bash
docker compose down -v
```

---

### Phương Án 2: Phát Triển Và Chạy Ứng Dụng Trực Tiếp Trên Local

Phương án này phù hợp khi bạn cần thay đổi mã nguồn liên tục, cần debug nhanh mà không muốn chờ đợi quá trình build lại Docker container của ứng dụng Go.

Bước 1: Chạy riêng các dịch vụ hạ tầng phụ thuộc bằng Docker Compose:
```bash
docker compose up -d postgres redis kafka
```

Bước 2: Nhân bản file cấu hình biến môi trường cục bộ:
```bash
cp .env.example .env
```
Lưu ý: Nếu thông tin kết nối dịch vụ local của bạn khác biệt so với file mẫu `.env.example`, bạn chỉ cần chỉnh sửa trực tiếp các thông số này trong file `.env` vừa tạo. File `.env` này sẽ được nạp tự động bằng `godotenv` lúc app bắt đầu chạy và nạp đè lên các giá trị cấu hình mặc định.

Bước 3: Tải thư viện phụ thuộc và khởi chạy ứng dụng Go:
```bash
go run cmd/server/main.go
```
Khi chạy thành công, mã nguồn Go local sẽ tự động kết nối vào Postgres, Redis và Kafka đang chạy trong container Docker, đồng thời tự động chạy migration và mở cổng API local tại `http://localhost:8080`.

### Tính Chống Chịu Lỗi Cao & Khởi Động Độc Lập (Graceful Fallback)

Đây là điểm thể hiện đẳng cấp thiết kế Senior/Tech Lead của Boilerplate này: **Tính chống chịu lỗi cao (Resilience) và Khả năng khởi động độc lập (Graceful Fallback)**. Ứng dụng vẫn có khả năng khởi động và phục vụ các yêu cầu API cơ bản ngay cả khi các dịch vụ không bắt buộc (Redis, Kafka) gặp sự cố hoặc không chạy kèm.

- **PostgreSQL (Bắt buộc)**: Nếu lỗi kết nối, ứng dụng sẽ log lỗi Fatal và dừng lại (vì không có DB thì sản phẩm mẫu không thể lưu trữ dữ liệu).
- **Redis (Không bắt buộc)**: Khi khởi động, gói `pkg/database/redis.go` sẽ thực hiện Ping để kiểm tra kết nối. Nếu không có Redis (lỗi kết nối), ứng dụng chỉ ghi nhận một cảnh báo (Warn log: `Redis connection failed. Features requiring cache may fail.`) nhưng không làm sụp ứng dụng. Biến con trỏ `redis` trong container được gán bằng `nil`.
- **Kafka (Không bắt buộc)**: Tương tự, nếu khởi tạo Kafka Producer thất bại, ứng dụng chỉ ghi log cảnh báo (Warn log: `Kafka Producer initialization failed. Event publishing disabled.`) nhưng không làm sụp ứng dụng. Biến con trỏ `producer` được gán bằng `nil`.
- **Điểm kiểm tra sức khỏe (/health)**: Hàm xử lý kiểm tra sức khỏe đã được viết sẵn điều kiện kiểm tra an toàn `if redis != nil`. Nếu không có Redis, hệ thống chỉ báo trạng thái `"redis": "down"` nhưng HTTP status code trả về vẫn là 200 OK.

#### Cách tinh chỉnh Docker Compose để chạy độc lập PostgreSQL và App

Nếu bạn muốn tắt hoàn toàn Redis và Kafka để tiết kiệm tài nguyên RAM/CPU cho máy tính local (chỉ chạy Postgres và App Go):

1. Mở file `docker-compose.yml` tìm đến phần cấu hình dịch vụ `app`.
2. Chỉnh sửa phần `depends_on` để chỉ phụ thuộc vào duy nhất `postgres`:
   ```yaml
   app:
     build:
       context: .
       dockerfile: Dockerfile
     container_name: boilerplate-app
     ports:
       - "8080:8080"
     depends_on:
       postgres:
         condition: service_healthy
   ```
3. Tiến hành comment hoặc xóa các block khai báo dịch vụ `redis` và `kafka` cùng phân vùng `redis_data` ở cuối file `docker-compose.yml`.
4. Chạy lệnh khởi động:
   ```bash
   docker compose up --build
   ```
   Ứng dụng Go khi khởi động sẽ tự nhận diện việc thiếu hụt Redis/Kafka, bỏ qua một cách an toàn và vận hành trơn tru với duy nhất PostgreSQL.

### Bộ Lệnh Makefile Tiện Ích

Một tệp `Makefile` tự động biên dịch và tạo hướng dẫn tích hợp sẵn để tối ưu hóa toàn bộ quá trình phát triển, kiểm thử và vận hành Docker Compose cho nhóm làm việc.

Chạy lệnh sau tại thư mục gốc để hiển thị toàn bộ hướng dẫn sử dụng:
```bash
make help
```

#### Các lệnh chính được hỗ trợ:
- `make run`: Khởi chạy ứng dụng Go local ở chế độ phát triển (Development).
- `make build`: Biên dịch mã nguồn tối ưu thành file nhị phân chạy production (tĩnh, lược bỏ symbols).
- `make test`: Chạy toàn bộ các bài kiểm thử unit tests của dự án.
- `make test-race`: Chạy toàn bộ unit tests kết hợp cờ kiểm tra lỗi xung đột luồng (`go test -race`).
- `make docker-up`: Khởi chạy toàn bộ hạ tầng (Postgres, Redis, Kafka) và ứng dụng Go bằng Docker Compose.
- `make docker-down`: Dừng toàn bộ các container đang chạy và xóa sạch dữ liệu bộ đệm (volumes).
- `make wire`: Tự động sinh mã nguồn giải quyết Dependency Injection qua Google Wire.
- `make migrate-create name=ten_migration`: Tạo nhanh một file SQL migration mới định dạng Goose trong thư mục `database/migrations/` gắn nhãn thời gian tự động.
- `make clean`: Dọn dẹp các tệp nhị phân biên dịch thừa và xóa bộ nhớ đệm cache kiểm thử local.

---

## Các Cơ Chế Kỹ Thuật Đặc Thù

### Quản Lý Lỗi Tập Trung (Domain Error Translator)
Để tuân thủ nghiêm ngặt nguyên lý SOLID, các Usecase nghiệp vụ không được trả về mã trạng thái HTTP (như 404 hay 400). Thay vào đó, chúng trả về các loại lỗi nghiệp vụ thuần túy thông qua gói `pkg/apperr`:
- `apperr.NewValidationError(...)` -> Tương ứng với HTTP 400 Bad Request
- `apperr.NewNotFoundError(...)` -> Tương ứng với HTTP 404 Not Found
- `apperr.NewConflictError(...)` -> Tương ứng với HTTP 409 Conflict
- `apperr.NewUnauthorizedError(...)` -> Tương ứng với HTTP 401 Unauthorized
- `apperr.NewForbiddenError(...)` -> Tương ứng với HTTP 403 Forbidden
- `apperr.NewInternalError(...)` -> Tương ứng với HTTP 500 Internal Server Error

Tại tầng biên (Gin Handler), gói `pkg/response` sẽ bắt lấy các lỗi AppError này, tự động quy đổi ra HTTP status phù hợp và trả về cấu trúc JSON đồng nhất:
```json
{
  "success": false,
  "error": {
    "code": "PRODUCT_SKU_EXISTS",
    "message": "product with SKU SKU123 already exists"
  }
}
```

### Bộ Log Cấu Trúc Tracing Với Zap
- Middleware Request ID: Tự động gán hoặc lan truyền header `X-Request-ID`. Mọi dòng log phát sinh trong suốt vòng đời của request đều chứa mã định danh này để thuận tiện tìm kiếm sau này.
- GORM Zap Logger: Tích hợp trực tiếp câu lệnh truy vấn SQL của GORM vào log Zap dưới dạng cấu trúc JSON, tự động đánh dấu cảnh báo (Warn) với các câu lệnh chạy chậm (Slow SQL >200ms).
- Middleware Recovery: Tự động bắt toàn bộ các sự cố nghiêm trọng (runtime panic) trong API handlers, log đầy đủ stack trace vào Zap dưới dạng JSON và trả về JSON 500 an toàn cho người dùng cuối thay vì làm dừng luồng dịch vụ.

### Bản Nhúng Database Migrations Bằng Goose
Quản lý lịch sử cơ sở dữ liệu bằng Goose mang lại trải nghiệm mượt mà:
- Toàn bộ thay đổi cơ cấu bảng được viết dưới dạng file `.sql` thuần túy đặt trong `database/migrations/`.
- Tiêu chuẩn viết file SQL migration bắt buộc phải có thẻ điều hướng UP và DOWN:
  ```sql
  -- +goose Up
  CREATE TABLE demo (...);

  -- +goose Down
  DROP TABLE demo;
  ```
- Nhờ công nghệ `go:embed` trong `database/migrations.go`, các file SQL này được nén và tích hợp trực tiếp vào file chạy nhị phân Go khi biên dịch. Nhờ đó, container chạy Production không cần phải mount thêm thư mục SQL ngoài.

---

## Hướng Dẫn Thêm Mới Một Module Nghiệp Vụ (Tuân Thủ SOLID)

Khi cần phát triển thêm một nghiệp vụ mới (ví dụ: `order` hoặc `customer`), hãy thực hiện theo đúng thứ tự thiết kế Clean Architecture sau:

1. Tạo thư mục module mới tại `internal/modules/[tên_module_mới]`.
2. Định nghĩa các thực thể và các interface trừu tượng trong thư mục `domain/`:
   - `domain/entity.go`: Khai báo struct thực thể, các ràng buộc và hàm khởi tạo kiểm tra invariants.
   - `domain/repository.go`: Khai báo interface tương tác cơ sở dữ liệu.
   - `domain/usecase.go`: Khai báo interface điều phối các tiến trình nghiệp vụ.
3. Triển khai logic nghiệp vụ thực tế trong thư mục `usecase/usecase.go` (lớp này chỉ phụ thuộc vào interface repository của Domain).
4. Triển khai code tương tác DB trong thư mục `repository/postgres_repository.go` bằng thư viện GORM.
5. Triển khai mã nguồn vận chuyển API trong thư mục `delivery/http/handler.go` sử dụng Gin.
6. Mở file container `internal/app/app.go` và tiến hành nối dây (wire) module mới này một cách thủ công trong hàm `NewApp`.

---

## Hướng Dẫn Kích Hoạt Google Wire Dự Phòng (DI Tự Động)

Nếu hệ thống phình to lên hàng trăm module và việc nối dây thủ công trong file `internal/app/app.go` trở nên quá dài và phức tạp:

Bước 1: Cài đặt công cụ CLI Google Wire trên máy cá nhân:
```bash
go install github.com/google/wire/cmd/wire@latest
```

Bước 2: Mở terminal tại thư mục chứa file wire cấu hình của dự án và chạy lệnh:
```bash
cd internal/app
wire
```
Hệ thống sẽ tự động quét tệp `internal/app/wire.go` đã được thiết lập sẵn các Provider Sets của dự án và sinh ra tệp liên kết tự động `wire_gen.go`.

Bước 3: Mở file chạy chính `cmd/server/main.go`, tìm dòng gọi hàm nối dây thủ công:
```go
// Tìm dòng này:
appInstance, err := app.NewApp(cfg, log)

// Và thay thế bằng dòng gọi hàm khởi tạo tự động của Google Wire:
appInstance, err := app.InitializeApp(cfg, log)
```
Sau đó chạy lệnh biên dịch dự án. Hệ thống sẽ vận hành hoàn toàn bằng cơ chế Dependency Injection tự động được giải quyết ở bước biên dịch compile-time.
