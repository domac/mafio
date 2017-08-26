package agent

import (
	"encoding/json"
	"github.com/domac/mafio/util"
	"io/ioutil"
	"path/filepath"
)

//配置选项
type Options struct {

	// 基本参数
	HTTPAddress         string `flag:"http-address"`
	MaxReadChannelSize  int    `flag:"max-read-channel-size"`
	MaxWriteChannelSize int    `flag:"max-write-channel-size"`
	MaxWriteBulkSize    int    `flag:"max-write-bulk-size"`
	AgentId             string `flag:"m-id"`
	AgentGroup          string `flag:"m-group"`
	Logger              Logger

	Input  string `flag:"input"`
	Output string `flag:"output"`
	Filter string `flag:"filter"`

	//插件参数
	FilePath     string `flag:"filepath"`
	InfluxdbAddr string `flag:"influxdb-addr"`

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
	for _, confPath := range pluginsConf {
		realPath, _ := filepath.Abs(confPath)
		if !util.IsExist(realPath) {
			self.Logger.Warnf("plugin config file didn't exist : %s", realPath)
			continue
		}

		b, err := ioutil.ReadFile(realPath)
		if err != nil {
			continue
		}
		content := make(map[string]interface{})
		err = json.Unmarshal(b, &content)

		if err != nil {
			continue
		}

		pluginName, ok := content["@pluginName"]
		if !ok {
			continue
		}
		key := pluginName.(string)
		delete(content, "@pluginName")
		self.PluginsConfigs[key] = content
	}

	return nil
}
