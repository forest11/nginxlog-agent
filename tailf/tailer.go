package tailf

import (
	"os"

	"github.com/hpcloud/tail"
)

// Follower describes an object that continuously emits a stream of lines
type Follower interface {
	Lines() chan *tail.Line
	OnError(func(error))
}

type followerImpl struct {
	filename string
	t        *tail.Tail
}

// NewFollower creates a new Follower instance for a given file (given by name)
func NewFollower(filename string) (Follower, error) {
	f := &followerImpl{
		filename: filename,
	}

	if err := f.start(); err != nil {
		return nil, err
	}

	return f, nil
}

// 获取单个文件的大小
func getSize(path string) int64 {
	fileInfo, err := os.Stat(path)
	if err != nil {
		panic(err)

	}
	fileSize := fileInfo.Size() //获取size
	return fileSize
}

// 从文件的最后开始读
func (f *followerImpl) start() error {
	fileSize := getSize(f.filename)
	seek := tail.SeekInfo{Offset: fileSize, Whence: 0}
	t, err := tail.TailFile(f.filename, tail.Config{
		Location: &seek,
		Follow:   true,
		ReOpen:   true,
		Poll:     true,
	})

	if err != nil {
		return err
	}

	f.t = t
	return nil
}

func (f *followerImpl) OnError(cb func(error)) {
	go func() {
		err := f.t.Wait()
		if err != nil {
			cb(err)
		}
	}()
}

func (f *followerImpl) Lines() chan *tail.Line {
	return f.t.Lines
}
