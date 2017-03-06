package input

import (
	"bytes"
	"encoding/json"
	"github.com/domac/mafio/util"
	"io/ioutil"
	"time"
)

//本地文件since信息数据库
//记录文件从哪里开始
type SinceDBInfo struct {
	Offset int64 `json:"offset,omitempty"`
}

//载入diskdb
func (self *FileInputService) LoadSinceDBInfos() (err error) {
	var (
		raw []byte
	)
	self.ctx.Logger().Debug("LoadSinceDBInfos")
	self.SinceDBInfos = map[string]*SinceDBInfo{}

	if self.SinceDBPath == "" || self.SinceDBPath == "/dev/null" {
		self.ctx.Logger().Warnf("No valid sincedb path")
		return
	}

	if !util.IsExist(self.SinceDBPath) {
		self.ctx.Logger().Debugf("sincedb not found: %q", self.SinceDBPath)
		return
	}

	if raw, err = ioutil.ReadFile(self.SinceDBPath); err != nil {
		self.ctx.Logger().Errorf("Read sincedb failed: %q\n%s", self.SinceDBPath, err)
		return
	}

	if err = json.Unmarshal(raw, &self.SinceDBInfos); err != nil {
		self.ctx.Logger().Errorf("Unmarshal sincedb failed: %q\n%s", self.SinceDBPath, err)
		return
	}

	return
}

//周期性地保存文件位置信息到磁盘db
func (self *FileInputService) SaveSinceDBInfos() (err error) {
	var (
		raw []byte
	)
	self.ctx.Logger().Debug("save file watch offset record")
	self.SinceDBLastSaveTime = time.Now()

	if self.SinceDBPath == "" || self.SinceDBPath == "/dev/null" {
		self.ctx.Logger().Warnf("No valid sincedb path")
		return
	}

	if raw, err = json.Marshal(self.SinceDBInfos); err != nil {
		self.ctx.Logger().Errorf("Marshal sincedb failed: %s", err)
		return
	}
	self.sinceDBLastInfosRaw = raw

	if err = ioutil.WriteFile(self.SinceDBPath, raw, 0664); err != nil {
		self.ctx.Logger().Errorf("Write sincedb failed: %q\n%s", self.SinceDBPath, err)
		return
	}

	return
}

func (self *FileInputService) CheckSaveSinceDBInfos() (err error) {
	var (
		raw []byte
	)
	if time.Since(self.SinceDBLastSaveTime) > time.Duration(self.SinceDBWriteInterval)*time.Second {
		if raw, err = json.Marshal(self.SinceDBInfos); err != nil {
			self.ctx.Logger().Errorf("Marshal sincedb failed: %s", err)
			return
		}
		if bytes.Compare(raw, self.sinceDBLastInfosRaw) != 0 {
			err = self.SaveSinceDBInfos()
		}
	}
	return
}

func (self *FileInputService) CheckSaveSinceDBInfosLoop() (err error) {
	for {
		time.Sleep(time.Duration(self.SinceDBWriteInterval) * time.Second)
		if err = self.CheckSaveSinceDBInfos(); err != nil {
			return
		}
	}
	return
}
