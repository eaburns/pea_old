package main

import (
	"fmt"
	unix "syscall"
	"unsafe"
)

const (
	readDirBufSize        = 8192
	openDirResultErrnoTag = 0
	openDirResultOKTag    = 1
	readDirResultErrnoTag = 0
	readDirResultEndTag   = 1
	readDirResultOKTag    = 2
	statResultErrnoTag    = 0
	statResultOKTag       = 1
)

type (
	Dir           = os_2Fposix__0__5FDir__
	OpenDirResult = os_2Fposix__0_OpenDirResult__
	ReadDirResult = os_2Fposix__0_ReadDirResult__
	StatResult    = os_2Fposix__0_StatResult__
	Stat          = os_2Fposix__0_Stat__
)

func F0_os_2Fposix__STDIN_5FFILENO__(ret *int)  { *ret = int(unix.Stdin) }
func F0_os_2Fposix__STDOUT_5FFILENO__(ret *int) { *ret = int(unix.Stdout) }
func F0_os_2Fposix__STDERR_5FFILENO__(ret *int) { *ret = int(unix.Stderr) }

func F0_os_2Fposix__EACCES__(ret *int)  { *ret = int(unix.EACCES) }
func F0_os_2Fposix__EEXIST__(ret *int)  { *ret = int(unix.EEXIST) }
func F0_os_2Fposix__EISDIR__(ret *int)  { *ret = int(unix.EISDIR) }
func F0_os_2Fposix__ENOENT__(ret *int)  { *ret = int(unix.ENOENT) }
func F0_os_2Fposix__ENOTDIR__(ret *int) { *ret = int(unix.ENOTDIR) }

func F0_os_2Fposix__strerror_3A__(errno int, ret *[]byte) {
	if errno < 0 {
		errno = -errno
	}
	*ret = []byte(unix.Errno(errno).Error())
}

func F0_os_2Fposix__O_5FRDONLY__(ret *int)    { *ret = unix.O_RDONLY }
func F0_os_2Fposix__O_5FWRONLY__(ret *int)    { *ret = unix.O_WRONLY }
func F0_os_2Fposix__O_5FRDWR__(ret *int)      { *ret = unix.O_RDWR }
func F0_os_2Fposix__O_5FAPPEND__(ret *int)    { *ret = unix.O_APPEND }
func F0_os_2Fposix__O_5FCREAT__(ret *int)     { *ret = unix.O_CREAT }
func F0_os_2Fposix__O_5FEXCL__(ret *int)      { *ret = unix.O_EXCL }
func F0_os_2Fposix__O_5FTRUNC__(ret *int)     { *ret = unix.O_TRUNC }
func F0_os_2Fposix__O_5FDIRECTORY__(ret *int) { *ret = unix.O_DIRECTORY }

func F0_os_2Fposix__open_3Amode_3Aperm_3A__(path *[]byte, mode int, perm int, ret *int) {
	cpath, ok := cstr(path)
	if !ok {
		*ret = -int(unix.EINVAL)
	}
retry:
	fd, _, e := unix.Syscall(unix.SYS_OPEN, cpath, uintptr(mode), uintptr(perm))
	switch {
	case e == unix.EINTR:
		goto retry
	case e != 0:
		*ret = -int(e)
	default:
		*ret = int(fd)
	}
}

func F0_os_2Fposix__close_3A__(fd int, ret *int) {
retry:
	_, _, e := unix.Syscall(unix.SYS_CLOSE, uintptr(fd), 0, 0)
	if e == unix.EINTR {
		goto retry
	}
	*ret = -int(e)
}

func F0_os_2Fposix__read_3Abuf_3A__(fd int, buf *[]byte, ret *int) {
	bufP := uintptr(unsafe.Pointer(&(*buf)[0]))
	bufLen := uintptr(len(*buf))
retry:
	n, _, e := unix.Syscall(unix.SYS_READ, uintptr(fd), bufP, bufLen)
	switch {
	case e == unix.EINTR:
		goto retry
	case e != 0:
		*ret = -int(e)
	default:
		*ret = int(n)
	}
}

func F0_os_2Fposix__write_3Abuf_3A__(fd int, buf *[]byte, ret *int) {
	bufP := uintptr(unsafe.Pointer(&(*buf)[0]))
	bufLen := uintptr(len(*buf))
retry:
	n, _, e := unix.Syscall(unix.SYS_WRITE, uintptr(fd), bufP, bufLen)
	switch {
	case e == unix.EINTR:
		goto retry
	case e != 0:
		*ret = -int(e)
	default:
		*ret = int(n)
	}
}

func F0_os_2Fposix__S_5FIFMT__(ret *uint)   { *ret = unix.S_IFMT }
func F0_os_2Fposix__S_5FIFBLK__(ret *uint)  { *ret = unix.S_IFBLK }
func F0_os_2Fposix__S_5FIFCHR__(ret *uint)  { *ret = unix.S_IFCHR }
func F0_os_2Fposix__S_5FIFIFO__(ret *uint)  { *ret = unix.S_IFIFO }
func F0_os_2Fposix__S_5FIFREG__(ret *uint)  { *ret = unix.S_IFREG }
func F0_os_2Fposix__S_5FIFDIR__(ret *uint)  { *ret = unix.S_IFDIR }
func F0_os_2Fposix__S_5FIFLNK__(ret *uint)  { *ret = unix.S_IFLNK }
func F0_os_2Fposix__S_5FIFSOCK__(ret *uint) { *ret = unix.S_IFSOCK }
func F0_os_2Fposix__S_5FIRWXU__(ret *uint)  { *ret = unix.S_IRWXU }
func F0_os_2Fposix__S_5FIRUSR__(ret *uint)  { *ret = unix.S_IRUSR }
func F0_os_2Fposix__S_5FIWUSR__(ret *uint)  { *ret = unix.S_IWUSR }
func F0_os_2Fposix__S_5FIXUSR__(ret *uint)  { *ret = unix.S_IXUSR }
func F0_os_2Fposix__S_5FIRWXG__(ret *uint)  { *ret = unix.S_IRWXG }
func F0_os_2Fposix__S_5FIRGRP__(ret *uint)  { *ret = unix.S_IRGRP }
func F0_os_2Fposix__S_5FIWGRP__(ret *uint)  { *ret = unix.S_IWGRP }
func F0_os_2Fposix__S_5FIXGRP__(ret *uint)  { *ret = unix.S_IXGRP }
func F0_os_2Fposix__S_5FIRWXO__(ret *uint)  { *ret = unix.S_IRWXO }
func F0_os_2Fposix__S_5FIROTH__(ret *uint)  { *ret = unix.S_IROTH }
func F0_os_2Fposix__S_5FIWOTH__(ret *uint)  { *ret = unix.S_IWOTH }
func F0_os_2Fposix__S_5FIXOTH__(ret *uint)  { *ret = unix.S_IXOTH }

func F0_os_2Fposix__fstat_3A__(fd int, ret *StatResult) {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		*ret = StatResult{
			tag:      statResultErrnoTag,
			errno_3A: int(err.(unix.Errno)),
		}
	} else {
		*ret = StatResult{
			tag: statResultOKTag,
			ok_3A: Stat{
				mode: uint(stat.Mode),
				size: stat.Size,
			},
		}
	}
}

func F0_os_2Fposix__unlink_3A__(path *[]byte, ret *int) {
	cpath, ok := cstr(path)
	if !ok {
		*ret = -int(unix.EINVAL)
	}
retry:
	_, _, e := unix.Syscall(unix.SYS_UNLINK, cpath, 0, 0)
	if e == unix.EINTR {
		goto retry
	}
	*ret = -int(e)
}

func F0_os_2Fposix__fdOpenDir_3A__(fd int, ret *OpenDirResult) {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		*ret = OpenDirResult{
			tag:      openDirResultErrnoTag,
			errno_3A: int(err.(unix.Errno)),
		}
		return
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		*ret = OpenDirResult{
			tag:      openDirResultErrnoTag,
			errno_3A: -int(unix.ENOTDIR),
		}
		return
	}
	*ret = OpenDirResult{
		tag:   openDirResultOKTag,
		ok_3A: &Dir{fd: int(fd), buf: make([]uint8, 0)},
	}
}

func F0_os_2Fposix__readDir_3A__(dir *Dir, ret *ReadDirResult) {
	if len(dir.buf) == 0 {
		dir.buf = make([]uint8, readDirBufSize)
	}
	if dir.p == dir.n {
		n, err := unix.ReadDirent(dir.fd, dir.buf)
		switch {
		case err != nil:
			e := err.(unix.Errno)
			fmt.Println("==>", err.Error())
			*ret = ReadDirResult{
				tag:      readDirResultErrnoTag,
				errno_3A: int(e),
			}
			return
		case n == 0:
			dir.buf = nil // free the buffer, since we are done.
			*ret = ReadDirResult{
				tag: readDirResultEndTag,
			}
			return
		default:
			dir.p = 0
			dir.n = int(n)
		}
	}
	names := make([]string, 0, 1)
	n, _, names := unix.ParseDirent(dir.buf[dir.p:dir.n], 1, names)
	dir.p += n
	*ret = ReadDirResult{
		tag:   readDirResultOKTag,
		ok_3A: []byte(names[0]),
	}
}

func F0_os_2Fposix__closeDir_3A__(dir *Dir, ret *int) {
retry:
	_, _, e := unix.Syscall(unix.SYS_CLOSE, uintptr(dir.fd), 0, 0)
	if e == unix.EINTR {
		goto retry
	}
	*ret = -int(e)
}

func F0_os_2Fposix__mkdir_3Aperm_3A__(path *[]byte, perm int, ret *int) {
	cpath, ok := cstr(path)
	if !ok {
		*ret = -int(unix.EINVAL)
	}
retry:
	_, _, e := unix.Syscall(unix.SYS_MKDIR, cpath, uintptr(perm), 0)
	if e == unix.EINTR {
		goto retry
	}
	*ret = -int(e)
}

func F0_os_2Fposix__rmdir_3A__(path *[]byte, ret *int) {
	cpath, ok := cstr(path)
	if !ok {
		*ret = -int(unix.EINVAL)
	}
retry:
	_, _, e := unix.Syscall(unix.SYS_RMDIR, cpath, 0, 0)
	if e == unix.EINTR {
		goto retry
	}
	*ret = -int(e)
}

func cstr(str *[]byte) (uintptr, bool) {
	cstr := make([]byte, len(*str)+1)
	for i, b := range *str {
		if b == 0 {
			return 0, false
		}
		cstr[i] = b
	}
	return uintptr(unsafe.Pointer(&cstr[0])), true
}
