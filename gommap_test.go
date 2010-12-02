package gommap_test


import (
    . "gocheck"
    "testing"
    "gommap"
    "path"
    "os"
    "io/ioutil"
)


func TestAll(t *testing.T) {
    TestingT(t)
}

type S struct{
    file *os.File
}

var _ = Suite(&S{})

var testData = []byte("0123456789ABCDEF")

func (s *S) SetUpTest(c *C) {
    testPath := path.Join(c.MkDir(), "test.txt")
    file, err := os.Open(testPath, os.O_RDWR | os.O_CREAT, 0644)
    if err != nil {
        panic(err.String())
    }
    s.file = file
    s.file.Write(testData)
}

func (s *S) TearDownTest(c *C) {
    s.file.Close()
}

func (s *S) TestUnsafeUnmap(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_SHARED)
    c.Assert(err, IsNil)
    c.Assert(mmap.UnsafeUnmap(), IsNil)
}

func (s *S) TestReadWrite(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_SHARED)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()
    c.Assert([]uint8(mmap), Equals, testData)

    mmap[9] = 'X'
    mmap.Sync(gommap.MS_SYNC)

    fileData, err := ioutil.ReadFile(s.file.Name())
    c.Assert(err, IsNil)
    c.Assert(fileData, Equals, []byte("012345678XABCDEF"))
}

func (s *S) TestSliceMethods(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_SHARED)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()
    c.Assert([]uint8(mmap), Equals, testData)

    mmap[9] = 'X'
    mmap[7:10].Sync(gommap.MS_SYNC)

    fileData, err := ioutil.ReadFile(s.file.Name())
    c.Assert(err, IsNil)
    c.Assert(fileData, Equals, []byte("012345678XABCDEF"))
}

func (s *S) TestProtFlagsAndErr(c *C) {
    testPath := s.file.Name()
    s.file.Close()
    file, err := os.Open(testPath, os.O_RDONLY, 0)
    c.Assert(err, IsNil)
    s.file = file
    _, err = gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                        gommap.MAP_SHARED)
    // For this to happen, both the error and the protection flag work.
    c.Assert(err, Equals, os.EACCES)
}

func (s *S) TestFlags(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_PRIVATE)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()

    mmap[9] = 'X'
    mmap.Sync(gommap.MS_SYNC)

    fileData, err := ioutil.ReadFile(s.file.Name())
    c.Assert(err, IsNil)
    // Shouldn't have written, since the map is private.
    c.Assert(fileData, Equals, []byte("0123456789ABCDEF"))
}

func (s *S) TestAdvise(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_PRIVATE)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()

    // A bit tricky to blackbox-test these.

    err = mmap.Advise(gommap.MADV_RANDOM)
    c.Assert(err, IsNil)

    err = mmap.Advise(9999)
    c.Assert(err, Matches, "invalid argument")
}

func (s *S) TestProtect(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ, gommap.MAP_SHARED)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()
    c.Assert([]uint8(mmap), Equals, testData)

    err = mmap.Protect(gommap.PROT_READ | gommap.PROT_WRITE)
    c.Assert(err, IsNil)

    // If this operation doesn't blow up tests, the call above worked.
    mmap[9] = 'X'
}


func (s *S) TestLock(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_PRIVATE)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()

    // A bit tricky to blackbox-test these.

    err = mmap.Lock()
    c.Assert(err, IsNil)

    err = mmap.Lock()
    c.Assert(err, IsNil)

    err = mmap.Unlock()
    c.Assert(err, IsNil)

    err = mmap.Unlock()
    c.Assert(err, IsNil)
}

func (s *S) TestInCoreUnderOnePage(c *C) {
    mmap, err := gommap.Map(s.file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_PRIVATE)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()

    mapped, err := mmap.InCore()
    c.Assert(err, IsNil)
    c.Assert(mapped, Equals, []uint8{1})
}

func (s *S) TestInCoreTwoPages(c *C) {
    testPath := path.Join(c.MkDir(), "test.txt")
    file, err := os.Open(testPath, os.O_RDWR | os.O_CREAT, 0644)
    c.Assert(err, IsNil)
    defer file.Close()

    file.Seek(int64(os.Getpagesize() * 2 - 1), 0)
    file.Write([]uint8{'x'})

    mmap, err := gommap.Map(file.Fd(), gommap.PROT_READ | gommap.PROT_WRITE,
                            gommap.MAP_PRIVATE)
    c.Assert(err, IsNil)
    defer mmap.UnsafeUnmap()

    // Not entirely a stable test, but should usually work.

    mmap[len(mmap)-1] = 'x'

    mapped, err := mmap.InCore()
    c.Assert(err, IsNil)
    c.Assert(mapped, Equals, []uint8{0, 1})

    mmap[0] = 'x'

    mapped, err = mmap.InCore()
    c.Assert(err, IsNil)
    c.Assert(mapped, Equals, []uint8{1, 1})
}
