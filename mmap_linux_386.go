package gommap

import "syscall"

func mmap_syscall(addr, length, prot, flags, fd uintptr, offset int64) (uintptr, uintptr) {
	page := uintptr(offset / 4096)
	if offset != int64(page)*4096 {
		return 0, syscall.EINVAL
	}
	addr, _, errno := syscall.Syscall6(syscall.SYS_MMAP2, addr, length, prot, flags, fd, page)
	return addr, errno
}
