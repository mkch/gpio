package gpiosysfs

import (
	"os"
)

type gpioFile struct {
	Path         string
	rfile, wfile *os.File
}

func (f *gpioFile) Close() (err error) {
	if f.rfile != nil {
		err = f.rfile.Close()
		if err != nil {
			return
		}
	}
	if f.wfile != nil {
		err = f.wfile.Close()
		if err != nil {
			return
		}
	}
	return
}

func (f *gpioFile) ReadAt0(p []byte) (n int, err error) {
	if f.rfile == nil {
		f.rfile, err = os.Open(f.Path)
		if err != nil {
			f.rfile = nil
			return
		}
	}
	return f.rfile.ReadAt(p, 0)
}

func (f *gpioFile) WriteAt0(p []byte) (n int, err error) {
	if f.wfile == nil {
		f.wfile, err = os.OpenFile(f.Path, os.O_WRONLY, 0)
		if err != nil {
			return
		}
	}
	return f.wfile.WriteAt(p, 0)
}
