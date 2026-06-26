package vfs

import "io/fs"

var (
	_ fs.FS         = (*Overlay)(nil)
	_ fs.ReadDirFS  = (*Overlay)(nil)
	_ fs.ReadFileFS = (*Overlay)(nil)
	_ fs.StatFS     = (*Overlay)(nil)

	_ fs.FS         = (*allowFS)(nil)
	_ fs.ReadDirFS  = (*allowFS)(nil)
	_ fs.ReadFileFS = (*allowFS)(nil)
	_ fs.StatFS     = (*allowFS)(nil)
)
