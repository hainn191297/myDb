# Tổng quan Storage Engine

Tài liệu này trình bày các khái niệm nền tảng dùng để xây dựng storage engine cho myDb, dựa trên ghi chú bạn cung cấp.

## 1. Mục tiêu của Storage Engine

- **Bền vững (Durability):** Dữ liệu phải tồn tại trên đĩa kể cả khi hệ thống gặp sự cố.
- **Hiệu năng:** Giảm tối đa số lần truy cập đĩa – vốn chậm hơn RAM hàng nghìn lần.
- **Tổ chức dữ liệu linh hoạt:** Hỗ trợ tốt các kiểu truy vấn (lookup, range, scan...).
- **Giao dịch & khôi phục:** Tích hợp cơ chế concurrency control, WAL, recovery.

## 2. Kiến trúc nhiều lớp

| Lớp | Vai trò |
| --- | --- |
| Record/Data Model | Biểu diễn tuple/row/field. |
| File Manager | Quản lý file vật lý cho bảng/index. |
| Page Manager | Cắt file thành page cố định (4–16 KB). |
| Buffer Manager | Cache page nóng trong RAM. |
| Access Methods | B+Tree, hash, heap… |
| Disk Storage | Sector → block → page trên thiết bị thật. |

## 3. Khái niệm đĩa & page

- **Sector:** đơn vị phần cứng (512 B hoặc 4 KB). DBMS hiếm khi thao tác trực tiếp.
- **Block (OS):** gom nhiều sector (4 KB, 8 KB). Hệ điều hành đọc/ghi theo block.
- **Page (DBMS):** đơn vị chính để nạp vào buffer pool, mọi chi phí truy vấn đều tính theo số page I/O.

## 4. Tổ chức file

| Kiểu | Ưu | Nhược |
| --- | --- | --- |
| Heap | Insert nhanh, đơn giản. | Lookup phải scan toàn bộ. |
| Sorted | Range query nhanh. | Insert/update tốn chi phí re-sort. |
| Hash | Lookup bằng equality cực nhanh. | Không hỗ trợ range query. |

## 5. Access Methods

- **B+Tree:** tìm kiếm/chèn/xóa O(log n), hỗ trợ range scan → phổ biến nhất.
- **Hash:** cực mạnh cho equality predicate nhưng không dùng được cho range.

## 6. Buffer Manager

Giữ page trong RAM, theo dõi pin count, dirty bit và replacement (LRU, Clock…). Mục tiêu: tăng hit ratio, giảm tối đa page fault.

## 7. WAL (Write-Ahead Logging)

Quy tắc ghi log trước khi flush page:

1. Append record thay đổi vào log trên đĩa.
2. Sau đó mới flush data page.

Nhờ WAL, hệ thống có thể redo/undo khi crash, đảm bảo ACID.

## 8. Quản lý không gian

- Page directory: tìm nhanh page theo ID.
- Free-space map: biết page nào còn chỗ trống.
- Extent: nhóm nhiều page để đọc tuần tự hiệu quả.
- Segment: đại diện logic cho table/index.

## 9. Clustered vs Non-Clustered Index

| Loại | Đặc điểm |
| --- | --- |
| Clustered | Dữ liệu vật lý sắp theo key; range query rất nhanh; mỗi bảng chỉ có 1. |
| Non-Clustered | Lưu key → con trỏ tới record/page; có thể nhiều cái; tối ưu lookup theo nhiều cột. |

## 10. Luồng xử lý ví dụ

`SELECT * FROM employee WHERE ssn = '123';`

1. Parser tạo AST; optimizer chọn B+Tree trên `ssn`.
2. Buffer manager nạp page chứa root B+Tree.
3. Duyệt cây tới leaf, lấy page chứa record.
4. Trả dữ liệu cho executor.
5. Nếu cập nhật: ghi WAL trước, rồi mới flush page.

## 11. Hướng mở rộng

- **Buffer Pool chi tiết:** page table, dirty tracking, chiến lược flush & thay thế.
- **WAL/Recovery:** undo/redo, thuật toán ARIES, checkpoint.
- **B+Tree mechanics:** định dạng node, split/merge, clustered vs non-clustered, chi phí join.
- **File organization:** so sánh ứng dụng thực tế và tiêu chí lựa chọn.
