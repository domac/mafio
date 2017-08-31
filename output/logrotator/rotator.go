package logrotator

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// TimeFormat is the default format used for the suffix date and time on each rotated log.
	TimeFormat = "2006-01"
)

// Options allows you to customize tne behavior of a RotatingWriter.
type Options struct {
	TimeFormat         string
	TimeFormatAsPrefix bool
	RotateDaily        bool
	Compress           bool
	MaximumSize        int64
}

// RotatingWriter is a io.Writer which wraps a *os.File, suitable for log rotation.
type RotatingWriter struct {
	sync.Mutex

	file        *os.File
	currentSize int64
	lastMod     time.Time

	opts Options
}

// NewWriter creates a new file and returns a rotating writer.
func NewWriter(filename string, opts *Options) (*RotatingWriter, error) {

	_, err := os.Stat(filename)
	if err != nil {
		os.Create(filename)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}

	return NewWriterFromFile(file, opts)
}

// NewWriterFromFile creates a rotating writer using the provided file as base.
//
// The caller must take care to not close the file it provides here, as the RotatingWriter
// will do it automatically when rotating.
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

// readCurrentSize reads the current size from the file
func (w *RotatingWriter) readMetadata() error {
	fi, err := w.file.Stat()
	if err != nil {
		return err
	}

	w.currentSize = fi.Size()
	// Get the last modification date, rounded down to midnight from that day.
	// The idea is, if we start the writer at say 4pm, we still want to
	// do the rotation at midnight the next day, and not 24h later at 4pm.
	// To do this, we get midnight from the day of the last modification date.
	//
	// Now, if the file is older than 1 day, it will be rotated right away, but that's
	// not really a problem.
	w.lastMod = getMidnightFromDate(fi.ModTime())

	return nil
}

// Write implements the io.Writer interface.
//
// NOTE(vincent): the rotating is not perfect when you want to rotate based on the maximum size.
// We won't rotate in the middle of a call to Write, so a call to Write will either end up in the original
// file or the new file if it needs to be rotated.
// That is to say, a call to Write will never split up the data from b into two different files.
// With this you can see that, given a big enough b, the original file CAN be significantly bigger than
// the requested maximum size.
// However, in practice it will never be a problem:
//  - the std package log will always write small buffers
//  - if you use your own logger, you should never use a big buffer anyway.
func (w *RotatingWriter) Write(b []byte) (int, error) {
	w.Lock()
	defer w.Unlock()

	// We wrote more than the maximum size - rotate
	if w.opts.MaximumSize > 0 && w.currentSize >= w.opts.MaximumSize {
		if err := w.rotateClear(); err != nil {
			return -1, err
		}
	}

	if w.opts.RotateDaily {
		now := time.Now()
		elapsed := now.Sub(w.lastMod)

		// Elapsed time is more than a day - rotate
		if elapsed > time.Hour*24 {
			if err := w.rotateClear(); err != nil {
				return -1, err
			}
		}
	}

	// otherwise we can just continue to write data

	n, err := w.file.Write(b)
	w.currentSize += int64(n)

	return n, err
}

func (w *RotatingWriter) rotateClear() error {
	original := w.file.Name()
	return os.Truncate(original, 0)
}

// rotate rotates the file. must be called while having the file lock
func (w *RotatingWriter) rotate() error {
	original := w.file.Name()

	// 1. close the original file
	if err := w.file.Close(); err != nil {
		return err
	}

	// 2. compute the destination filename.
	// For example, if:
	//  - original is /tmp/mylog.log
	//  - lastMod is 2016-05-01 20:00 (local time)
	//  - opts is empty
	// then destName will be /tmp/mylog.log.2016-05-01_2000
	destName := makeDestName(original, w.lastMod, &w.opts)
	_, err := os.Stat(destName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 3. rename the original file.
	if err := os.Rename(original, destName); err != nil {
		return err
	}

	// 4. if compression is needed, do it in a goroutine.
	if w.opts.Compress {
		go func() {
			if err := w.compressFile(destName); err != nil {
				fmt.Printf("unable to compress file '%s'. err=%v\n", destName, err)
				return
			}

			// no error to compress the data and to rename it
			// to its last filename, we can now safely remove
			// the original uncompressed file.
			if err := os.Remove(destName); err != nil {
				fmt.Printf("unable to remove file '%s'. err=%v\n", destName, err)
			}
		}()
	}

	// 5. reset the last mod date.
	//
	// We know for sure we're the following day at this point,
	// so it's safe to assume time.Now() will return the date from the following day.
	// We reset the last mod date to the following day, midnight.
	w.lastMod = getMidnightFromDate(time.Now())

	// 6. open a new file at the original path.
	file, err := os.OpenFile(original, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return err
	}

	// 7. reset the file and current size in the writer.
	w.file = file
	w.currentSize = 0

	return nil
}

// compressFile compresses the file at destName into a file at destName.gz
func (w *RotatingWriter) compressFile(destName string) (err error) {
	var (
		rotated *os.File
		tmpFile *os.File
	)

	// 1. open the rotated file.
	if rotated, err = os.Open(destName); err != nil {
		return
	}
	defer rotated.Close()

	// 2. gzip the file to a temporary file.
	if tmpFile, err = w.gzip(rotated); err != nil {
		return err
	}

	// 3. force close just before renaming
	rotated.Close()
	tn := tmpFile.Name()
	tmpFile.Close()

	// 4. rename the gzipped file.
	return os.Rename(tn, destName+".gz")
}

func (w *RotatingWriter) gzip(src *os.File) (f *os.File, err error) {
	dir := filepath.Dir(src.Name())

	// 1. create a tmp file which will be the rotated one but compressed.
	//
	// NOTE(vincent): we NEED to create the file in the same dir as the original !
	// Otherwise we risk getting errors while renaming, because the underlying syscall
	// won't permit cross-device renaming. That is, if you have your tempdir (typically /tmp on a Unix)
	// on a particular device, and your log directory on another, the rename won't work.
	if f, err = ioutil.TempFile(dir, "tmp"); err != nil {
		return nil, err
	}

	// 2. compression
	gw := gzip.NewWriter(f)
	defer gw.Close()

	_, err = io.Copy(gw, src)

	return
}

// makeDestName transforms the name by integrating the startDate, formatted according to the options provided.
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
