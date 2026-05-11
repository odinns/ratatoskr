package ratatoskr

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type fileIdentity struct {
	device uint64
	inode  uint64
}

type pathSize struct {
	allocated int64
	apparent  int64
}

func identityOf(info os.FileInfo) (fileIdentity, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, false
	}
	return fileIdentity{device: uint64(stat.Dev), inode: uint64(stat.Ino)}, true
}

func fileSizeOf(info os.FileInfo) pathSize {
	apparent := info.Size()
	allocated := apparent
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Blocks > 0 {
		allocated = stat.Blocks * 512
	}
	return pathSize{allocated: allocated, apparent: apparent}
}

func sizeOf(path string) (pathSize, error) {
	var total pathSize
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size := fileSizeOf(info)
			total.allocated += size.allocated
			total.apparent += size.apparent
		}
		return nil
	})
	return total, err
}
