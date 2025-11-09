# myDb

_Xem phiên bản tiếng Anh tại [README.md](README.md)._ 

**myDb** là một dự án học tập nhằm tự tay xây dựng cơ sở dữ liệu quan hệ bằng Go từ những nguyên lý đầu tiên. Kho lưu trữ này ghi lại toàn bộ hành trình—từ lưu trữ dạng log, chỉ mục B+Tree cho đến phân tích cú pháp SQL, giao dịch và công cụ vận hành. Mọi thứ được giữ đơn giản để dễ mày mò và phục vụ mục đích giáo dục.

> Trạng thái: giai đoạn thiết kế ban đầu. Tham khảo `architecture.md` để xem sơ đồ các lớp.

## Mục tiêu dự án

- Hiểu cách một SQL engine vận hành trọn vẹn (parser → planner → executor → storage).
- Tự triển khai các cam kết giao dịch (WAL, khóa, mức cô lập) thay vì dựa vào thư viện có sẵn.
- Vững vàng hơn với các mẫu lập trình hệ thống bằng Go: goroutine, channel, quản lý bộ nhớ, chẩn đoán.
- Ghi lại các đánh đổi và bài học để việc mở rộng trong tương lai.

## Tính năng dự kiến

| Khu vực | Phạm vi |
| --- | --- |
| Client / SQL | API gRPC, lexer/parser SQL, planner dựa trên rule, kết quả dạng streaming |
| Transactions | Quản lý phiên, khóa hai pha cho ghi, mức cô lập cấu hình được, retry/timeout |
| Storage Engine | KV store có WAL, chỉ mục B+Tree, buffer pool + page cache, checkpoint/chính sách fsync |
| Ops & Tooling | Cấu hình qua etcd, giới hạn tốc độ, metrics, tracing, logging có cấu trúc |

Chi tiết xem `architecture.md`, tài liệu mô tả vòng đời request và vai trò từng lớp.

## Bắt đầu

1. Cài Go ≥ 1.22 và đảm bảo `$GOPATH/bin` nằm trong `PATH`.
2. Sao chép repo:

   ```bash
   git clone https://github.com/hainn191297/myDb.git
   cd myDb
   ```

3. Khởi tạo module và chạy test đầu tiên:

   ```bash
   go mod tidy
   go test ./...
   ```

Khi storage engine hoàn thiện dần, tài liệu sẽ cập nhật hướng dẫn build/run cụ thể.

## Quy trình phát triển

- Ưu tiên lát cắt dọc: thêm tính năng nhỏ end-to-end (ví dụ WAL append → test khôi phục) trước khi chuyển việc.
- Viết unit test cho từng package (`parser`, `planner`, `storage/wal`, `txn/locks`) và giữ tốc độ chạy nhanh.
- Dùng benchmark Go (`go test -bench . ./...`) để phát hiện regression khi cấu trúc dữ liệu thay đổi.
- Ghi TODO trong code và phản chiếu ý tưởng lớn hơn ở roadmap để tránh lan man phạm vi.

## Lộ trình

1. KV nhúng tối thiểu với log append-only + chỉ mục trong bộ nhớ.
2. WAL + định dạng page + bộ quản lý buffer cùng chính sách fsync.
3. Chỉ mục B+Tree và hỗ trợ range scan.
4. Lexer/parser SQL và các toán tử executor cơ bản (scan, filter, project, insert/update/delete).
5. Trình quản lý giao dịch với khóa + retry và giao thức gRPC cho client.
6. Lớp quan sát (metrics, tracing, logging) và cấu hình qua etcd.
7. Mục tiêu mở rộng: cải tiến planner, replication/failover, thử nghiệm lưu trữ dạng cột.

## Tài liệu

- `architecture.md`: sơ đồ tầng + giải thích chi tiết về client interface, hệ thống giao dịch, storage engine và vận hành.
- `docs/` (dự kiến): chuyên đề về định dạng WAL, khóa, buffer manager và quy trình vận hành.

## Tài nguyên học tập

- *Database Internals* – Alex Petrov: giải thích sâu về storage engine và consensus.
- *Designing Data-Intensive Applications* – Martin Kleppmann: nền tảng giao dịch và hệ phân tán.
- *Fundamentals of Database Systems* – Elmasri & Navathe: giáo trình kinh điển về mô hình dữ liệu và thiết kế SQL.
- Bài giảng CMU 15-445/645 (Intro to Database Systems) cho các mẫu triển khai cụ thể.

## Đóng góp

Dự án hiện phục vụ mục đích học tập cá nhân, nhưng mọi phản hồi và thảo luận đều được hoan nghênh. Hãy tạo issue để đặt câu hỏi/góp ý, hoặc fork repo để thử thiết kế khác và chia sẻ kết quả.
