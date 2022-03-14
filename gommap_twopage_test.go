// +build !windows,!freebsd

package gommap

import (
	"os"
	"path"

	. "launchpad.net/gocheck"
)

func (s *S) TestIsResidentTwoPages(c *C) {
	testPath := path.Join(c.MkDir(), "test.txt")
	file, err := os.Create(testPath)
	c.Assert(err, IsNil)
	defer file.Close()

	file.Seek(int64(os.Getpagesize()*2-1), 0)
	file.Write([]byte{'x'})

	mmap, err := Map(file.Fd(), PROT_READ|PROT_WRITE, MAP_PRIVATE)
	c.Assert(err, IsNil)
	defer mmap.UnsafeUnmap()

	// Not entirely a stable test, but should usually work.

	mmap[len(mmap)-1] = 'x'

	mapped, err := mmap.IsResident()
	c.Assert(err, IsNil)
	c.Assert(mapped, DeepEquals, []bool{false, true})

	mmap[0] = 'x'

	mapped, err = mmap.IsResident()
	c.Assert(err, IsNil)
	c.Assert(mapped, DeepEquals, []bool{true, true})
}
