package logrotator

import (
	"compress/gzip"
	"fmt"
	"github.com/domac/mafio/util"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	TimeFormat = "2006-01"
)

type Options struct {
	TimeFormat         string
	TimeFormatAsPrefix bool
	RotateDaily        bool
	Compress           bool
	MaximumSize        int64
}

type RotatingWriter struct {
	sync.Mutex

	file        *os.File
	currentSize int64
	lastMod     time.Time

	opts Options
}

func NewWriter(filename string, opts *Options) (*RotatingWriter, error) {

	_, err := os.Stat(filename)
	if err != nil {
		dir := filepath.Dir(filename)
		util.ShellRun("mkdir -p " + dir)
		util.ShellRun("touch  " + filename)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}

	return NewWriterFromFile(file, opts)
}

func NewWriterFromFile(file *os.File, opts *Options) (*RotatingWriter, error) {
	w := &RotatingWriter{file: file}

	if opts != nil {
		w.opts.TimeFormat = opts.TimeFormat
		w.opts.TimeFormatAsPrefix = opts.TimeFormatAsPrefix
		w.opts.RotateDaily = opts.RotateDaily
		w.opts.Compress = opts.Compress
		w.opts.MaximumSize = opts.MaximumSize
	}

	if err := w.readMetadata(); err != nil {
		return nil, fmt.Errorf("unable to read current file size. err=%v", err)
	}

	return w, nil
}

func getMidnightFromDate(t time.Time) time.Time {
	yy, mm, dd := t.Date()
	return time.Date(yy, mm, dd, 0, 0, 0, 0, t.Location())
}

func (w *RotatingWriter) readMetadata() error {
	fi, err := w.file.Stat()
	if err != nil {
		return err
	}

	w.currentSize = fi.Size()
	w.lastMod = getMidnightFromDate(fi.ModTime())

	return nil
}

func (w *RotatingWriter) Write(b []byte) (int, error) {
	w.Lock()
	defer w.Unlock()

	if w.opts.MaximumSize > 0 && w.currentSize >= w.opts.MaximumSize {
		if err := w.rotateClear(); err != nil {
			return -1, err
		}
	}

	if w.opts.RotateDaily {
		now := time.Now()
		elapsed := now.Sub(w.lastMod)

		if elapsed > time.Hour*24 {
			if err := w.rotateClear(); err != nil {
				return -1, err
			}
		}
	}

	n, err := w.file.Write(b)
	w.currentSize += int64(n)

	return n, err
}

func (w *RotatingWriter) rotateClear() error {
	original := w.file.Name()
	w.currentSize = 0
	return os.Truncate(original, 0)
}

func (w *RotatingWriter) rotate() error {
	original := w.file.Name()

	if err := w.file.Close(); err != nil {
		return err
	}

	destName := makeDestName(original, w.lastMod, &w.opts)
	_, err := os.Stat(destName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(original, destName); err != nil {
		return err
	}

	if w.opts.Compress {
		go func() {
			if err := w.compressFile(destName); err != nil {
				fmt.Printf("unable to compress file '%s'. err=%v\n", destName, err)
				return
			}

			if err := os.Remove(destName); err != nil {
				fmt.Printf("unable to remove file '%s'. err=%v\n", destName, err)
			}
		}()
	}

	w.lastMod = getMidnightFromDate(time.Now())

	file, err := os.OpenFile(original, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	w.file = file
	w.currentSize = 0

	return nil
}

func (w *RotatingWriter) compressFile(destName string) (err error) {
	var (
		rotated *os.File
		tmpFile *os.File
	)

	if rotated, err = os.Open(destName); err != nil {
		return
	}
	defer rotated.Close()

	if tmpFile, err = w.gzip(rotated); err != nil {
		return err
	}

	rotated.Close()
	tn := tmpFile.Name()
	tmpFile.Close()

	return os.Rename(tn, destName+".gz")
}

func (w *RotatingWriter) gzip(src *os.File) (f *os.File, err error) {
	dir := filepath.Dir(src.Name())

	if f, err = ioutil.TempFile(dir, "tmp"); err != nil {
		return nil, err
	}

	// 2. compression
	gw := gzip.NewWriter(f)
	defer gw.Close()

	_, err = io.Copy(gw, src)

	return
}

func makeDestName(name string, startDate time.Time, opts *Options) string {
	tf := TimeFormat
	if opts.TimeFormat != "" {
		tf = opts.TimeFormat
	}

	if opts.TimeFormatAsPrefix {
		ext := filepath.Ext(name)
		name = name[:len(name)-len(ext)]

		return name + "." + startDate.Format(tf) + ext
	}

	return name + "." + startDate.Format(tf)
}
