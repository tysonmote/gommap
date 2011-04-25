package gommap

import "syscall"

func mmap_syscall(addr, length, prot, flags, fd uintptr, offset int64) (uintptr, uintptr) {
	addr, _, errno := syscall.Syscall6(syscall.SYS_MMAP, addr, length, prot, flags, fd, uintptr(offset))
	return addr, errno
}
