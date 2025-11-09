package wal

import "io"

// Writer describes the write-ahead log append interface.
type Writer interface {
    Append(record []byte) error
    Sync() error
}

// Reader replays records for recovery.
type Reader interface {
    Next() ([]byte, error)
}

// FileWAL is a placeholder implementation that will eventually write to disk.
type FileWAL struct {
    writer io.Writer
}

func NewFileWAL(w io.Writer) *FileWAL {
    return &FileWAL{writer: w}
}

func (w *FileWAL) Append(record []byte) error {
    _, err := w.writer.Write(record)
    return err
}

func (w *FileWAL) Sync() error {
    // TODO: use fsync once os.File is wired.
    return nil
}
