package agent

import (
	"encoding/json"
	"errors"
	"github.com/domac/mafio/util"
	"io/ioutil"
	"path/filepath"
	"strings"
)

//配置选项
type Options struct {

	// 基本参数
	HTTPAddress         string `flag:"http-address"`
	MaxReadChannelSize  int    `flag:"max-read-channel-size"`
	MaxWriteChannelSize int    `flag:"max-write-channel-size"`
	MaxWriteBulkSize    int    `flag:"max-write-bulk-size"`
	SendInterval        int    `flag:"send-interval"`
	AgentId             string `flag:"m-id"`
	AgentGroup          string `flag:"m-group"`
	Logger              Logger

	Input  string `flag:"input"`
	Output string `flag:"output"`
	Filter string `flag:"filter"`

	//插件参数
	InfluxdbAddr string `flag:"influxdb-addr"`
	FormatStr    string `flag:"f"`

	//插件配置数据
	PluginsConfigs map[string]map[string]interface{}
	ConfigFilePath string
}

func NewOptions(configFilePath string) *Options {
	return &Options{
		HTTPAddress:         "0.0.0.0:10630",
		AgentId:             "localhost",
		AgentGroup:          "devops",
		MaxWriteChannelSize: 4096,
		MaxWriteBulkSize:    500,
		Logger:              defaultLogger,
		ConfigFilePath:      configFilePath,
		PluginsConfigs:      make(map[string]map[string]interface{}),
	}
}

//加载插件的配置数据
func (self *Options) LoadPluginsConf(pluginsConf []string) error {

	if self.ConfigFilePath == "" {
		return nil
	}

	cfp, _ := filepath.Abs(self.ConfigFilePath)
	//配置文件所在目录信息
	dir := filepath.Dir(cfp)

	if !util.IsExist(dir) {
		return errors.New("config path not exist")
	}

	for _, confPath := range pluginsConf {
		//获取相对路径信息
		confPath = strings.TrimSpace(confPath)
		realPath := ""
		if strings.HasPrefix(confPath, "/") {
			realPath, _ = filepath.Abs(confPath)
		} else {
			realPath = filepath.Join(dir, confPath)
		}

		if !util.IsExist(realPath) {
			self.Logger.Warnf("plugin config file didn't exist : %s", realPath)
			continue
		}
		//刷新配置
		err := self.flushConfig(realPath)
		if err != nil {
			continue
		}
	}

	return nil
}

//读取指定了路径的文件,把内容刷新到全局配置映射中
func (self *Options) flushConfig(realPath string) error {
	b, err := ioutil.ReadFile(realPath)
	if err != nil {
		return err
	}
	content := make(map[string]interface{})
	err = json.Unmarshal(b, &content)
	if err != nil {
		return err
	}
	//获取匹配的插件名称
	pluginName, ok := content["@pluginName"]
	if !ok {
		//如果没有配置插件名称, 则任务配置不合法
		return errors.New("config file didn't include @pluginName: " + realPath)
	}
	key := pluginName.(string)
	//delete(content, "@pluginName")
	self.PluginsConfigs[key] = content
	return nil
}
