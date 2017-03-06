package main

//*************************************************************
//
//
// 这里只是程序的引导代码,具体处理直接看 agentd.go 的 Main()方法
//
// Too Long , Don't Read !
//*************************************************************

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/domac/mafio/agent"
	"github.com/domac/mafio/register"
	"github.com/domac/mafio/version"
	"github.com/judwhite/go-svc/svc"
	"github.com/mreiferson/go-options"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

//程序启动参数
var (
	flagSet = flag.NewFlagSet("mafio", flag.ExitOnError)

	showVersion         = flagSet.Bool("version", false, "print version string")                                         //版本
	httpAddress         = flagSet.String("http-address", "0.0.0.0:10630", "<addr>:<port> to listen on for HTTP clients") //http定义地址
	config              = flagSet.String("config", "", "path to config file")
	MaxReadChannelSize  = flagSet.Int("max-read-channel-size", 4096, "max readChannel size")
	MaxWriteChannelSize = flagSet.Int("max-write-channel-size", 4096, "max writeChannel size")
	MaxWriteBulkSize    = flagSet.Int("max-write-bulk-size", 500, "max writeBulk size")

	etcdEndpoint = flagSet.String("etcd-endpoint", "0.0.0.0:2379", "ectd service discovery address")
	AgentId      = flagSet.String("agent-id", "sky01", "the service name which ectd can find it")
	AgentGroup   = flagSet.String("agent-group", "net01", "the service group which agent work on")
	Input        = flagSet.String("input", "stdin", "input plugin")
	Outout       = flagSet.String("output", "stdout", "output plugin")
	Filter       = flagSet.String("filter", "valid", "filter plugin")
	FilePath     = flagSet.String("filepath", "", "use for file watch")
	InfluxDBAddr = flagSet.String("influxdb-addr", "", "influxDB addr to metrics")
)

//程序封装
type program struct {
	Agentd *agent.Agentd
}

//框架初始化
func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		//切换工作目录
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

//Agent的参数使用说明
//go run main.go -h 展示的内容
func agentUsage() {
	fmt.Printf("%s\n", version.Show())
	flagSet.PrintDefaults()
}

//程序启动
func (p *program) Start() error {

	flagSet.Usage = agentUsage

	flagSet.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(version.Verbose("mafioAget"))
		os.Exit(0)
	}

	fmt.Println(version.Show())

	var cfg map[string]interface{}
	if *config != "" {
		_, err := toml.DecodeFile(*config, &cfg)
		if err != nil {
			log.Fatalf("ERROR: failed to load config file %s - %s", *config, err.Error())
		}
	}

	opts := agent.NewOptions()
	options.Resolve(opts, flagSet, cfg)

	//初始化插件注册
	register.Init()

	//后台进程创建
	daemon := agent.New(opts)
	daemon.Main()
	p.Agentd = daemon
	return nil
}

//程序停止
func (p *program) Stop() error {
	if p.Agentd != nil {
		p.Agentd.Exit()
	}
	return nil
}

//引导程序
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	prg := &program{}
	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}
