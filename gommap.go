// Type representing memory mapped file or device.  The Data field gives direct
// access to the memory mapped content.
//
// IMPORTANT NOTE (1): The Data field in MMap types is backed by an unsafe
// memory region, which is not covered by the normal rules of Go's memory
// management. If a slice is taken out of it, and then the memory is explicitly
// unmapped through one of the available methods, both the Data field and the
// slice obtained will now silently point to invalid memory.  Attempting to
// access data in them will crash the application.
package gommap

import (
    "syscall"
    "reflect"
    "unsafe"
    "os"
)


// Type representing memory mapped file or device.  The Data field gives direct
// access to the memory mapped content.
//
// IMPORTANT: Please see note (1) in the package documentation regarding the way
// in which the Data field behaves.
type MMap struct {
    Data []uint8
    internalData []uint8
}

// Create a new mapping in the virtual address space of the calling process.
// This function will attempt to map the entire file by using the fstat system
// call with the provided file descriptor to discover its length.
func Map(fd int, prot, flags uint) (*MMap, os.Error) {
    mmap, err := MapAt(0, fd, 0, -1, prot, flags)
    return mmap, err
}

// Create a new mapping in the virtual address space of the calling process,
// using the specified region of the provided file or device. If -1 is provided
// as length, this function will attempt to map until the end of the provided
// file descriptor by using the fstat system call to discover its length.
func MapRegion(fd int, offset, length int64,
               prot, flags uint) (*MMap, os.Error) {
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
           prot, flags uint) (*MMap, os.Error) {
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
    mmap := &MMap{}

    dh := (*reflect.SliceHeader)(unsafe.Pointer(&mmap.Data))
    dh.Data = addr
    dh.Len = int(length) // Hmmm.. truncating here feels like trouble.
    dh.Cap = dh.Len
    mmap.internalData = mmap.Data
    return mmap, nil
}

// Delete the entire mapped region allocation, flushing any remaining
// changes, if necessary.  Using the Data field or any slices taken out of it
// after this method is called will crash the application.
//
// IMPORTANT: Please see note (1) in the package documentation regarding
// details about the way in which the Data field behaves.
func (mmap *MMap) UnsafeUnmap() os.Error {
    err := mmap.UnsafeUnmapSlice(mmap.Data[:])
    mmap.internalData = nil
    mmap.Data = nil
    return err
}

// Delete the memory mapped region allocation as specified by the provided
// slice, which must have been obtained from the Data field.  This will also
// flush any remaining changes, if necessary.
func (mmap *MMap) UnsafeUnmapSlice(region []uint8) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MUNMAP,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Flush changes made to the memory mapped slice back to the device.  Without
// calling this method, there are no guarantees that changes will be flushed
// back before the region is unmapped.  The flags parameter specifies whether
// flushing should be done synchronously (before the method returns) with
// MS_SYNC, or asynchronously (flushing is just scheduled) with MS_ASYNC. 
func (mmap *MMap) Sync(flags uint) {
    mmap.SyncSlice(mmap.Data[:], flags)
}

// Flush changes made to the region determined by the provided memory mapped
// slice, which should have been taken out of the Data field, back to the
// device.  Without calling this method, there are no guarantees that changes
// will be flushed back before the region is unmapped.  The flags parameter
// specifies whether flushing should be done synchronously (before the method
// returns) with MS_SYNC, or asynchronously (flushing is just scheduled) with
// MS_ASYNC.
func (mmap *MMap) SyncSlice(region []uint8, flags uint) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MSYNC,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(flags))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Advise the kernel about how to handle the mapped memory region in terms
// of input/output paging within the whole mapped memory region.
func (mmap *MMap) Advise(advice uint) os.Error {
    return mmap.AdviseSlice(mmap.Data[:], advice)
}

// Advise the kernel about how to handle the mapped memory region in terms
// of input/output paging within the provided memory region, which must be
// a slice of the Data field.
func (mmap *MMap) AdviseSlice(region []uint8, advice uint) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MADVISE,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(advice))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Change protection flags for the whole memory mapped region.
func (mmap *MMap) Protect(prot uint) os.Error {
    return mmap.ProtectSlice(mmap.Data[:], prot)
}

// Change protection flags for the memory region defined by the provided
// slice, which should be obtained from the Data field.
func (mmap *MMap) ProtectSlice(region []uint8, prot uint) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MPROTECT,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(prot))
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Lock all the mapped region into memory, preventing them from being
// swapped out.
func (mmap *MMap) Lock() os.Error {
    return mmap.LockSlice(mmap.Data[:])
}

// Lock the mapped region defined by the provided slice into memory,
// preventing them from being swapped out.  The slice must have been
// created out of the Data field.
func (mmap *MMap) LockSlice(region []uint8) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MLOCK,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Unlock all the memory mapped region, allowing it to swap out again.
func (mmap *MMap) Unlock() os.Error {
    return mmap.UnlockSlice(mmap.Data[:])
}

// Unlock the mapped region defined by the provided slice, allowing it to
// swap out again. The slice must have been created out of the Data field.
func (mmap *MMap) UnlockSlice(region []uint8) os.Error {
    if err := mmap.checkSlice(region); err != nil {
        return err
    }
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    _, _, errno := syscall.Syscall(syscall.SYS_MUNLOCK,
                                   uintptr(rh.Data), uintptr(rh.Len), 0)
    if errno != 0 {
        return os.Errno(errno)
    }
    return nil
}

// Return an array of uint8 values informing with the lowest bit whether
// the given page in the mapped region was or not in memory at the time
// the call was made. Note that the higher bits are reserved for future
// use, so do not simply run an equality test with 1.
func (mmap *MMap) InCore() ([]uint8, os.Error) {
    return mmap.InCoreSlice(mmap.Data[:])
}

// Return an array of uint8 values informing with the lowest bit whether
// the given page in the mapped region was or not in memory at the time
// the call was made.  The region parameter informs the region to query,
// and should be aligned at a page boundary.  Note that the higher bits
// are reserved for future use, so do not simply run an equality test
// with 1.
func (mmap *MMap) InCoreSlice(region []uint8) ([]uint8, os.Error) {
    if err := mmap.checkSlice(region); err != nil {
        return nil, err
    }
    pageSize := os.Getpagesize()
    result := make([]uint8, (len(region) + pageSize - 1) / pageSize)
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    resulth := *(*reflect.SliceHeader)(unsafe.Pointer(&result))
    _, _, errno := syscall.Syscall(syscall.SYS_MINCORE,
                                   uintptr(rh.Data), uintptr(rh.Len),
                                   uintptr(resulth.Data))
    if errno != 0 {
        return nil, os.Errno(errno)
    }
    return result, nil
}

func (mmap *MMap) checkSlice(region []uint8) os.Error {
    rh := *(*reflect.SliceHeader)(unsafe.Pointer(&region))
    dh := *(*reflect.SliceHeader)(unsafe.Pointer(&mmap.internalData))
    if rh.Data < dh.Data || rh.Data + uintptr(rh.Len) > dh.Data + uintptr(dh.Len) {
        return os.NewError("Region must be a slice of mmap.Data")
    }
    return nil
}
