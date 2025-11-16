1. Mục tiêu của cơ chế lưu trữ trong DBMS

Mọi DBMS đều cần một storage engine để:

Lưu trữ dữ liệu trên đĩa (disk) một cách bền vững (durability)

Truy xuất dữ liệu nhanh dù đĩa là thiết bị rất chậm

Tổ chức dữ liệu sao cho tối ưu cho các kiểu truy vấn khác nhau

Hỗ trợ giao dịch – concurrency – recovery

Điều này dẫn đến kiến trúc lưu trữ với một số tầng (layers).

2. Kiến trúc tổng quát của Storage Engine

Một DBMS không truy cập ổ đĩa trực tiếp. Nó dùng nhiều lớp:

Record / Data Model: dữ liệu biểu diễn thành record, field, row

File Manager: quản lý file vật lý

Page Manager: quản lý page (thường 4KB–16KB)

Buffer Manager: cache page trong RAM

Access Methods: B+Tree, Hash, Heap File

Disk Storage: sector → block → page

3. Các khái niệm nền tảng của bộ nhớ ổ đĩa
3.1. Sector – đơn vị nhỏ nhất của đĩa (hardware)

Thường 512 bytes hoặc 4096 bytes (Advanced Format)

DBMS không thao tác trực tiếp với sector.

3.2. Block (OS)

Hệ điều hành gom nhiều sector thành block (4KB, 8KB…).

DBMS đọc/ghi theo block.

3.3. Page (DBMS)

DBMS sử dụng page (frame) là đơn vị truy cập chính.

Thường 4KB–16KB — tùy hệ quản trị

Toàn bộ dữ liệu (table, index...) nằm trong các page.

Page là đơn vị nạp vào buffer pool.

→ Đây là lý do mọi thuật toán trong DBMS tính chi phí theo số lượng page I/O, như trong chương 19 của sách (chi phí truy vấn) .

4. Các kiểu tổ chức file (File Organization)

DBMS có nhiều cách tổ chức file vật lý, mỗi cách phù hợp một loại truy vấn:

4.1. Heap File (unsorted file)

Dữ liệu lưu theo thứ tự chèn.

Tìm kiếm tốn full scan.

Tối ưu cho insert nhanh.

Cấu trúc phổ biến trong mọi database.

4.2. Sorted File

Lưu trữ dữ liệu đã được sắp xếp theo khóa.

Tối ưu cho range query

Chi phí insert/update cao vì phải sắp lại file.

4.3. Hash File

Dùng hàm hash → bucket → record

Tối ưu cho truy vấn lookup bằng equality

Không phù hợp cho range queries.

5. Access Methods (B+Tree, Hash Index)

Để tăng tốc độ truy xuất, DBMS sử dụng cấu trúc dữ liệu chuyên dụng:

B+Tree Index

Phổ biến nhất

Tìm kiếm, chèn, xóa: O(log n)

Hỗ trợ range scan

Hash Index

Tìm kiếm bằng equality rất nhanh

Không hỗ trợ range scan

Trong sách (chương về query processing), B+Tree và hash được dùng làm access paths để giảm cost của join, select, project… .

6. Buffer Manager (cực kỳ quan trọng)

DBMS giữ nhiều page trong buffer pool (RAM).
Vì RAM nhanh hơn disk hàng nghìn lần, mọi thao tác như join/select đều cố gắng:

Giảm page I/O

Tăng page hit ratio

Dùng Least Recently Used (LRU) hoặc Clock replacement

Trong phần cost model của chương 19, bạn thấy mọi chi phí được tính bằng số page đọc/ghi vào buffer .

7. WAL – Write Ahead Log (cơ chế Durable & Recovery)

Khi cập nhật dữ liệu, DBMS:

Ghi log (WAL) trước vào disk

Ghi page thật sau

WAL đảm bảo:

Nếu crash → DBMS có thể undo / redo từ log

Tính durability của ACID

8. Cơ chế phân mảnh & tổ chức dữ liệu

Nhiều DBMS hỗ trợ:

Page directory: tra cứu nhanh vị trí page

Free-space map: quản lý page trống

Extent: nhóm nhiều page lại để tăng hiệu suất đọc tuần tự

Segment: file logic (table, index)

9. Index và Data nằm riêng (Clustered vs Non-clustered Index)
Clustered index

Dữ liệu sắp theo thứ tự index

Tối ưu range query

Mỗi table chỉ có 1 clustered index

Non-clustered index

Lưu cặp (key → pointer to record/page)

Tối ưu lookup nhiều cột

10. Tổng quan: Storage Engine hoạt động như thế nào?

Giả sử truy vấn:

SELECT * FROM EMPLOYEE WHERE Ssn = '123';


Flow:

SQL Parser → Query Plan

Optimizer chọn index (ví dụ B+Tree)

Buffer Manager nạp page chứa root của B+Tree

Truy theo cây → tìm leaf page

Đọc page chứa record

Trả kết quả

Nếu update → ghi WAL trước, rồi flush page sau

Tất cả các bước đều được mô tả chi tiết trong các chương 19 và 20 .
