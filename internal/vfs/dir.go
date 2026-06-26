package vfs

import (
	"io"
	"io/fs"
	"time"
)

type dirFile struct {
	entries []fs.DirEntry
	info    fs.FileInfo
	name    string
	offset  int
}

func newDirFile(name string, info fs.FileInfo, entries []fs.DirEntry) *dirFile {
	if info == nil {
		info = directoryInfo{name: name}
	}

	return &dirFile{name: name, info: info, entries: entries}
}

func (file *dirFile) Close() error {
	return nil
}

func (file *dirFile) Read(_ []byte) (int, error) {
	return 0, pathError("read", file.name, fs.ErrInvalid)
}

func (file *dirFile) ReadDir(count int) ([]fs.DirEntry, error) {
	if count <= 0 {
		remaining := file.entries[file.offset:]
		file.offset = len(file.entries)
		return remaining, nil
	}

	if file.offset >= len(file.entries) {
		return nil, io.EOF
	}

	end := min(file.offset+count, len(file.entries))
	entries := file.entries[file.offset:end]
	file.offset = end

	return entries, nil
}

func (file *dirFile) Stat() (fs.FileInfo, error) {
	return file.info, nil
}

type directoryInfo struct {
	name string
}

func (info directoryInfo) IsDir() bool {
	return true
}

func (info directoryInfo) ModTime() time.Time {
	return time.Time{}
}

func (info directoryInfo) Mode() fs.FileMode {
	return fs.ModeDir | 0o555
}

func (info directoryInfo) Name() string {
	if info.name == "." {
		return "."
	}

	return info.name
}

func (info directoryInfo) Size() int64 {
	return 0
}

func (info directoryInfo) Sys() any {
	return nil
}
