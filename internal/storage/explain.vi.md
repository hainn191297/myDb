# Giải thích luồng Buffer Pool

Sơ đồ PlantUML trong `buffer_flow.puml` mô tả cách buffer manager tương tác với các cấu trúc nội bộ và tầng đĩa của storage engine. Tài liệu này giải thích từng giai đoạn để bạn dễ đối chiếu khi đọc sơ đồ.

## Thành phần & vai trò

- **Client Query** – luồng thực thi/truy vấn yêu cầu page, đánh dấu dirty và trả page.
- **Buffer Manager (BM)** – kết hợp TableManager (mở file dữ liệu) và WAL Manager (mở log) để cung cấp buffer pool cho từng bảng.
- **Buffer Pool (BP)** – điều phối lookup, cấp phát frame, theo dõi dirty/pin count và điều khiển thay thế; mỗi buffer pool ghi thêm WAL trước khi flush xuống FileManager.
- **Page Table (PT)** – ánh xạ `PageID -> frameID` để cache hit không phải đọc đĩa.
- **Frame / Slot** – vùng chứa page cùng `pinCnt` và cờ `Dirty`.
- **Free List** – tập frame trống được cấp phát sẵn chỉ dùng cho `capacity` lần miss đầu tiên; khi dùng hết thì danh sách này trống hẳn.
- **LRU List** – cấu trúc thay thế sắp xếp frame theo mức độ sử dụng gần nhất.
- **Ứng viên thay thế (Victim)** – chính là frame (và page trong đó) bị chọn để giải phóng khỏi buffer pool khi cần chỗ trống.
- **Page Manager (PM)** – trừu tượng hóa layout file và thực hiện I/O hệ điều hành.
- **Disk** – các data file đảm bảo tính bền vững.

## Giai đoạn Request Page

1. **GetPage** – client gọi `GetPage(schema, table, PageID)` vào Buffer Manager; BM tái sử dụng/khởi tạo buffer pool bằng cách mở FileManager + WAL logger tương ứng, sau đó BP truy vấn page table.
2. **Cache Hit** – PT trả về frameID, BP tăng `pinCnt` để chặn eviction và trả page cho client.
3. **Cache Miss** – không có mapping nên BP cần frame mới:
   - Nếu free list vẫn còn frame cấp sẵn thì pop ra dùng slot đó.
   - Nếu đã hết, BP lấy ứng viên thay thế từ LRU: lặp qua các frame ít dùng nhất cho tới khi gặp frame chưa bị pin; frame bị loại sẽ được tái sử dụng ngay (không trả về free list).
4. **Ứng viên dirty** – nếu frame (victim) được chọn đang dirty, BP flush page đó qua PM rồi mới gỡ mapping khỏi page table và tái sử dụng frame.
5. **Đọc từ đĩa** – BP yêu cầu FileManager của bảng đọc page tương ứng (OS read block).
6. **Khởi tạo frame** – nạp dữ liệu vào frame, đặt `pinCnt = 1`, `Dirty = false`, chèn frame vào đầu LRU và thêm mapping mới trong page table, sau đó trả page cho client.

## Giai đoạn Update

- Khi client chỉnh sửa page trong bộ nhớ, nó gọi `MarkDirty` qua BP. Chỉ cờ dirty đổi trạng thái, page vẫn nằm trong cache.

## Giai đoạn Release

- Client gọi `Unpin(PageID)` khi xong việc; BP giảm `pinCnt`.
- Nếu về 0, frame trở thành ứng viên eviction và được đẩy dần về cuối LRU; frame còn pin vẫn nằm gần đầu.

## Giai đoạn Background Flush

- Các tiến trình nền hoặc lời gọi `FlushTable` sẽ duyệt tất cả frame dirty, ghi WAL trước, sync WAL, rồi mới flush page xuống FileManager → Disk để đảm bảo thứ tự WAL trước dữ liệu.

Luồng này giúp buffer manager cân bằng giữa tốc độ (cache hit), tính đúng đắn (pin count, dirty tracking) và độ bền (flush qua page manager), đồng thời thể hiện rõ trách nhiệm của từng cấu trúc.

## Lớp Heap Table

Sơ đồ `heap_flow.puml` mô tả cách heap table dùng Buffer Manager/WAL:

- Insert quét các page hiện có, khởi tạo định dạng slotted-page nếu cần, chèn bản ghi (payload key/value), append WAL rồi đánh dấu frame dirty.
- Delete chỉ việc đặt độ dài slot về 0 để có thể tái sử dụng slot trong tương lai; phần dữ liệu cũ sẽ bị ghi đè khi cần.
- Scan duyệt từng page/slot, copy dữ liệu rồi trả cho tầng cao hơn.
