package agent

//配置选项
type Options struct {

	// 基本参数
	HTTPAddress         string `flag:"http-address"`
	MaxReadChannelSize  int    `flag:"max-read-channel-size"`
	MaxWriteChannelSize int    `flag:"max-write-channel-size"`
	MaxWriteBulkSize    int    `flag:"max-write-bulk-size"`
	EtcdEndpoint        string `flag:"etcd-endpoint"`
	AgentId             string `flag:"agent-id"`
	AgentGroup          string `flag:"agent-group"`
	Logger              Logger

	Input  string `flag:"input"`
	Output string `flag:"output"`
	Filter string `flag:"filter"`

	//插件参数
	FilePath string `flag:"filepath"`

	InfluxdbAddr string `flag:"influxdb-addr"`
}

func NewOptions() *Options {
	return &Options{
		HTTPAddress:         "0.0.0.0:10630",
		EtcdEndpoint:        "0.0.0.0:2379",
		AgentId:             "localhost",
		AgentGroup:          "devops",
		MaxWriteChannelSize: 4096,
		MaxWriteBulkSize:    500,
		Logger:              defaultLogger,
	}
}
