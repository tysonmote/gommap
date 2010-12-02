// Type representing memory mapped file or device.  The returned MMap value,
// which is itself an alias to a []byte slice, gives direct access to the
// memory mapped content.
//
// IMPORTANT NOTE (1): The MMap type is backed by an unsafe memory region,
// which is not covered by the normal rules of Go's memory management. If a
// slice is taken out of it, and then the memory is explicitly unmapped through
// one of the available methods, both the MMap value itself and the slice
// obtained will now silently point to invalid memory.  Attempting to access
// data in them will crash the application.
package gommap

import (
    "syscall"
    "reflect"
    "unsafe"
    "os"
)


// Type representing memory mapped file or device.  The slice gives direct
// access to the memory mapped content.
//
// IMPORTANT: Please see note (1) in the package documentation regarding the way
// in which this type behaves.
type MMap []uint8

// Create a new mapping in the virtual address space of the calling process.
// This function will attempt to map the entire file by using the fstat system
// call with the provided file descriptor to discover its length.
func Map(fd int, prot, flags uint) (MMap, os.Error) {
    mmap, err := MapAt(0, fd, 0, -1, prot, flags)
    return mmap, err
}

// Create a new mapping in the virtual address space of the calling process,
// using the specified region of the provided file or device. If -1 is provided
// as length, this function will attempt to map until the end of the provided
// file descriptor by using the fstat system call to discover its length.
func MapRegion(fd int, offset, length int64,
               prot, flags uint) (MMap, os.Error) {
    mmap, err := MapAt(0, fd, offset, length, prot, flags)
    return mmap, err
}

// Create a new mapping in the virtual address space of the calling process,
// using the specified region of the provided file or device. The provided addr
// parameter will be used as a hint of the address where the kernel should
// position the memory mapped region. If -1 is provided as length, this
// function will attempt to map until the end of the provided file descriptor
// by using the fstat system call to discover its length.
func MapAt(addr uintptr, fd int, offset, length int64,
           prot, flags uint) (MMap, os.Error) {
    if length == -1 {
        var stat syscall.Stat_t
        if errno := syscall.Fstat(fd, &stat); errno != 0 {
            return nil, os.Errno(errno)
        }
        length = stat.Size
    }
    addr, _, errno := syscall.Syscall6(syscall.SYS_MMAP, addr,
                                       uintptr(length), uintptr(prot),
                                       uintptr(flags), uintptr(fd),
                                       uintptr(offset))
    if errno != 0 {
        return nil, os.Errno(errno)
    }
    mmap := MMap{}

    dh := (*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    dh.Data = addr
    dh.Len = int(length) // Hmmm.. truncating here feels like trouble.
    dh.Cap = dh.Len
    return mmap, nil
}

// Delete the memory mapped region defined by the mmap slice. This will also
// flush any remaining changes, if necessary.  Using mmap after this method
// has been called will crash the application.
//
// IMPORTANT: Please see note (1) in the package documentation regarding the way
// in which this type behaves.
func (mmap MMap) UnsafeUnmap() os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MUNMAP,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Flush changes made to the region determined by the mmap slice back to
// the device.  Without calling this method, there are no guarantees that
// changes will be flushed back before the region is unmapped.  The flags
// parameter specifies whether flushing should be done synchronously (before
// the method returns) with MS_SYNC, or asynchronously (flushing is just
// scheduled) with MS_ASYNC.
func (mmap MMap) Sync(flags uint) os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MSYNC,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(flags))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Advise the kernel about how to handle the mapped memory region in terms
// of input/output paging within the memory region defined by the mmap slice.
func (mmap MMap) Advise(advice uint) os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MADVISE,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(advice))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Change protection flags for the memory mapped region defined by
// the mmap slice.
func (mmap MMap) Protect(prot uint) os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MPROTECT,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(prot))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Lock the mapped region defined by the mmap slice, preventing it from
// being swapped out.
func (mmap MMap) Lock() os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MLOCK,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Unlock the mapped region defined by the mmap slice, allowing it to
// swap out again.
func (mmap MMap) Unlock() os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    _, _, errno := syscall.Syscall(syscall.SYS_MUNLOCK,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Return an array of uint8 values informing with the lowest bit whether
// the given page in the mapped region defined by the mmap slice was or
// not in memory at the time the call was made. Note that the higher bits
// are reserved for future use, so do not simply run an equality test
// with 1.
func (mmap MMap) InCore() ([]uint8, os.Error) {
    pageSize := os.Getpagesize()
    result := make([]uint8, (len(mmap) + pageSize - 1) / pageSize)
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap))
    resulth := *(*reflect.SliceHeader)(unsafe.Pointer(&result))
    _, _, errno := syscall.Syscall(syscall.SYS_MINCORE,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(resulth.Data))
    if errno != 0 {
        return nil, os.Errno(errno)
    }
    return result, nil
}
