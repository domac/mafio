package input

import (
	"bufio"
	"bytes"
	"errors"
	a "github.com/domac/mafio/agent"
	"github.com/go-fsnotify/fsnotify"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ModuleName = "file"

//文件输入服务
type FileInputService struct {
	ctx *a.Context

	Path                 string                  `json:"path"`
	StartPos             string                  `json:"start_position,omitempty"` // one of ["beginning", "end"]
	SinceDBPath          string                  `json:"sincedb_path,omitempty"`
	SinceDBWriteInterval int                     `json:"sincedb_write_interval,omitempty"`
	hostname             string                  `json:"-"`
	SinceDBInfos         map[string]*SinceDBInfo `json:"-"`
	sinceDBLastInfosRaw  []byte                  `json:"-"`
	SinceDBLastSaveTime  time.Time               `json:"-"`
}

func New() *FileInputService {
	return &FileInputService{}
}

func (self *FileInputService) SetContext(ctx *a.Context) {
	self.ctx = ctx
	self.StartPos = "end"
	self.SinceDBPath = "/tmp/sincedb.json"
	self.SinceDBWriteInterval = 15
	self.SinceDBInfos = map[string]*SinceDBInfo{}
}

//开启文件监听
func (self *FileInputService) StartInput() {

	configMap, ok := self.ctx.Agentd.GetOptions().PluginsConfigs[ModuleName]
	if !ok {
		self.ctx.Logger().Errorln("could't load config")
	}

	self.Path = configMap["stdFilePath"].(string)

	self.ctx.Logger().Infof("plugins input filepath %s", self.Path)

	if self.Path == "" {
		self.ctx.Logger().Errorln("file input fail: no file path found")
		os.Exit(2)
		return
	}

	var (
		matches []string
		fi      os.FileInfo
		err     error
	)

	//载入disk数据库
	if err = self.LoadSinceDBInfos(); err != nil {
		return
	}

	if matches, err = filepath.Glob(self.Path); err != nil {
		self.ctx.Logger().Errorf("gob (%s) failed", self.Path)
		return
	}

	go self.CheckSaveSinceDBInfosLoop()

	for _, fpath := range matches {
		if fpath, err = filepath.EvalSymlinks(fpath); err != nil {
			self.ctx.Logger().Errorf("Get symlinks failed: %q\n%v", fpath, err)
			continue
		}

		if fi, err = os.Stat(fpath); err != nil {
			self.ctx.Logger().Errorf("stat(%q) failed\n%s", self.Path, err)
			continue
		}

		if fi.IsDir() {
			self.ctx.Logger().Infof("Skipping directory: %q", self.Path)
			continue
		}

		readEventChan := make(chan fsnotify.Event, 10)
		//文件读入
		go self.fileReadLoop(readEventChan, fpath)
		//文件事件监听
		go self.fileWatchLoop(readEventChan, fpath, fsnotify.Create|fsnotify.Write)
	}
}

//文件读入
func (self *FileInputService) fileReadLoop(
	readEventChan chan fsnotify.Event,
	fpath string,
) (err error) {
	var (
		since     *SinceDBInfo
		fp        *os.File
		truncated bool
		ok        bool
		whence    int
		reader    *bufio.Reader
		line      string
		size      int

		buffer = &bytes.Buffer{}
	)

	if fpath, err = filepath.EvalSymlinks(fpath); err != nil {
		self.ctx.Logger().Errorf("Get symlinks failed: %q\n%v", fpath, err)
		return
	}

	if since, ok = self.SinceDBInfos[fpath]; !ok {
		self.SinceDBInfos[fpath] = &SinceDBInfo{}
		since = self.SinceDBInfos[fpath]
	}

	if since.Offset == 0 {
		if self.StartPos == "end" {
			whence = os.SEEK_END
		} else {
			whence = os.SEEK_SET
		}
	} else {
		whence = os.SEEK_SET
	}

	if fp, reader, err = openfile(fpath, since.Offset, whence); err != nil {
		return
	}
	defer fp.Close()

	if truncated, err = isFileTruncated(fp, since); err != nil {
		return
	}
	if truncated {
		self.ctx.Logger().Warnf("File truncated, seeking to beginning: %q", fpath)
		since.Offset = 0
		if _, err = fp.Seek(since.Offset, os.SEEK_SET); err != nil {
			self.ctx.Logger().Errorf("seek file failed: %q", fpath)
			return
		}
	}

	for {
		if line, size, err = readline(reader, buffer); err != nil {
			if err == io.EOF {
				watchev := <-readEventChan
				//self.ctx.Logger().Debug("fileReadLoop recv:", watchev)
				if watchev.Op&fsnotify.Create == fsnotify.Create {
					self.ctx.Logger().Warnf("File recreated, seeking to beginning: %q", fpath)
					fp.Close()
					since.Offset = 0
					if fp, reader, err = openfile(fpath, since.Offset, os.SEEK_SET); err != nil {
						return
					}
				}
				if truncated, err = isFileTruncated(fp, since); err != nil {
					return
				}
				if truncated {
					self.ctx.Logger().Warnf("File truncated, seeking to beginning: %q", fpath)
					since.Offset = 0
					if _, err = fp.Seek(since.Offset, os.SEEK_SET); err != nil {
						self.ctx.Logger().Errorf("seek file failed: %q", fpath)
						return
					}
					continue
				}
				//self.ctx.Logger().Debugf("watch %q %q %v", watchev.Name, fpath, watchev)
				continue
			} else {
				return
			}
		}

		since.Offset += int64(size)

		self.ctx.Agentd.Inchan <- []byte(line)
		self.CheckSaveSinceDBInfos()
	}
}

//文件操作事件监听
func (self *FileInputService) fileWatchLoop(readEventChan chan fsnotify.Event, fpath string, op fsnotify.Op) (err error) {
	var (
		event fsnotify.Event
	)
	for {
		if event, err = waitWatchEvent(fpath, op); err != nil {
			return
		}
		readEventChan <- event
	}
	return
}

func isFileTruncated(fp *os.File, since *SinceDBInfo) (truncated bool, err error) {
	var (
		fi os.FileInfo
	)
	if fi, err = fp.Stat(); err != nil {
		err = errors.New("stat file failed: " + fp.Name())
		return
	}
	if fi.Size() < since.Offset {
		truncated = true
	} else {
		truncated = false
	}
	return
}

func openfile(fpath string, offset int64, whence int) (fp *os.File, reader *bufio.Reader, err error) {
	if fp, err = os.Open(fpath); err != nil {
		err = errors.New("open file failed: " + fpath)
		return
	}

	if _, err = fp.Seek(offset, whence); err != nil {
		err = errors.New("seek file failed: " + fpath)
		return
	}

	reader = bufio.NewReaderSize(fp, 16*1024)
	return
}

func readline(reader *bufio.Reader, buffer *bytes.Buffer) (line string, size int, err error) {
	var (
		segment []byte
	)

	for {
		if segment, err = reader.ReadBytes('\n'); err != nil {
			if err != io.EOF {
				err = errors.New("read line failed")
			}
			return
		}

		if _, err = buffer.Write(segment); err != nil {
			err = errors.New("write buffer failed")
			return
		}

		if isPartialLine(segment) {
			time.Sleep(1 * time.Second)
		} else {
			size = buffer.Len()
			line = buffer.String()
			buffer.Reset()
			line = strings.TrimRight(line, "\r\n")
			return
		}
	}

	return
}

func isPartialLine(segment []byte) bool {
	if len(segment) < 1 {
		return true
	}
	if segment[len(segment)-1] != '\n' {
		return true
	}
	return false
}

var (
	mapWatcher = map[string]*fsnotify.Watcher{}
)

func waitWatchEvent(fpath string, op fsnotify.Op) (event fsnotify.Event, err error) {
	var (
		fdir    string
		watcher *fsnotify.Watcher
		ok      bool
	)

	if fpath, err = filepath.EvalSymlinks(fpath); err != nil {
		err = errors.New("Get symlinks failed: " + fpath)
		return
	}

	fdir = filepath.Dir(fpath)

	if watcher, ok = mapWatcher[fdir]; !ok {
		if watcher, err = fsnotify.NewWatcher(); err != nil {
			err = errors.New("create new watcher failed: " + fdir)
			return
		}
		mapWatcher[fdir] = watcher
		if err = watcher.Add(fdir); err != nil {
			err = errors.New("add new watch path failed: " + fdir)
			return
		}
	}

	for {
		select {
		case event = <-watcher.Events:
			if event.Name == fpath {
				if op > 0 {
					if event.Op&op > 0 {
						return
					}
				} else {
					return
				}
			}
		case err = <-watcher.Errors:
			err = errors.New("watcher error")
			return
		}
	}
	return
}
