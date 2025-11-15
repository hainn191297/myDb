# Giải thích luồng Buffer Pool

Sơ đồ PlantUML trong `flow.puml` mô tả cách buffer manager tương tác với các cấu trúc nội bộ và tầng đĩa của storage engine. Tài liệu này giải thích từng giai đoạn để bạn dễ đối chiếu khi đọc sơ đồ.

## Thành phần & vai trò

- **Client Query** – luồng thực thi/truy vấn yêu cầu page, đánh dấu dirty và trả page.
- **Buffer Pool (BP)** – điều phối lookup, cấp phát frame, theo dõi dirty/pin count và điều khiển thay thế.
- **Page Table (PT)** – ánh xạ `PageID -> frameID` để cache hit không phải đọc đĩa.
- **Frame / Slot** – vùng chứa page cùng `pinCnt` và cờ `Dirty`.
- **Free List** – danh sách frame trống; hết frame trống thì phải đi theo LRU để tìm ứng viên thay thế.
- **LRU List** – cấu trúc thay thế sắp xếp frame theo mức độ sử dụng gần nhất.
- **Ứng viên thay thế (Victim)** – chính là frame (và page trong đó) bị chọn để giải phóng khỏi buffer pool khi cần chỗ trống.
- **Page Manager (PM)** – trừu tượng hóa layout file và thực hiện I/O hệ điều hành.
- **Disk** – các data file đảm bảo tính bền vững.

## Giai đoạn Request Page

1. **GetPage** – client gọi `GetPage(PageID)`, BP truy vấn page table.
2. **Cache Hit** – PT trả về frameID, BP tăng `pinCnt` để chặn eviction và trả page cho client.
3. **Cache Miss** – không có mapping nên BP cần frame mới:
   - Nếu free list còn phần tử thì pop ra dùng.
   - Nếu đã đầy, BP lấy ứng viên thay thế từ LRU: lặp qua các frame ít dùng nhất cho tới khi gặp frame chưa bị pin; frame đang pin thì bỏ qua.
4. **Ứng viên dirty** – nếu frame (victim) được chọn đang dirty, BP flush page đó qua PM rồi mới gỡ mapping khỏi page table.
5. **Đọc từ đĩa** – BP yêu cầu PM đọc page tương ứng (OS read block).
6. **Khởi tạo frame** – nạp dữ liệu vào frame, đặt `pinCnt = 1`, `Dirty = false`, chèn frame vào đầu LRU và thêm mapping mới trong page table, sau đó trả page cho client.

## Giai đoạn Update

- Khi client chỉnh sửa page trong bộ nhớ, nó gọi `MarkDirty` qua BP. Chỉ cờ dirty đổi trạng thái, page vẫn nằm trong cache.

## Giai đoạn Release

- Client gọi `Unpin(PageID)` khi xong việc; BP giảm `pinCnt`.
- Nếu về 0, frame trở thành ứng viên eviction và được đẩy dần về cuối LRU; frame còn pin vẫn nằm gần đầu.

## Giai đoạn Background Flush

- Các tiến trình nền hoặc cơ chế theo ngưỡng có thể duyệt LRU để flush sớm các page dirty lạnh.
- Những frame dirty và không bị pin sẽ được ghi xuống đĩa qua PM → Disk để các lần eviction sau không phải chờ ghi.

Luồng này giúp buffer manager cân bằng giữa tốc độ (cache hit), tính đúng đắn (pin count, dirty tracking) và độ bền (flush qua page manager), đồng thời thể hiện rõ trách nhiệm của từng cấu trúc.
