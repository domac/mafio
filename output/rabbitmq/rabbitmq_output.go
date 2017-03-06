package rabbitmq

import (
	"errors"
	"github.com/bitly/go-hostpool"
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
	"github.com/streadway/amqp"
	"time"
)

const ModuleName = "rabbitmq"

type RabbitmqOutputService struct {
	ctx *a.Context

	URLs               []string
	Key                string
	Exchange           string
	ExchangeType       string
	ExchangeDurable    bool
	ExchangeAutoDelete bool
	Retries            int
	ReconnectDelay     int
	hostPool           hostpool.HostPool
	amqpClients        map[string]amqpClient
	isCheck            bool
}

type amqpClient struct {
	client    *amqp.Channel
	reconnect chan hostpool.HostPoolResponse
}

type amqpConn struct {
	Channel    *amqp.Channel
	Connection *amqp.Connection
}

func New() *RabbitmqOutputService {
	service := &RabbitmqOutputService{}
	return service
}

func (self *RabbitmqOutputService) SetContext(ctx *a.Context) {
	self.initClient()
	self.ctx = ctx
}

func options2map(opt *a.Options) map[string]interface{} {
	return nil
}

func (self *RabbitmqOutputService) initClient() {

	self.ctx.Logger().Println("start opening rabbitmq connection")
	ctxopt := self.ctx.Agentd.GetOptions()
	opt := options2map(ctxopt)

	if _, ok := opt["urls"]; ok {
		p_urls := opt["urls"].([]interface{})
		urls := []string{}
		for i := 0; i < len(p_urls); i++ {
			urls = append(urls, p_urls[i].(string))
		}
		self.URLs = urls
	}

	if _, ok := opt["exchange"]; ok {
		self.Exchange = opt["exchange"].(string)
	}

	if _, ok := opt["exchange_type"]; ok {
		self.ExchangeType = opt["exchange_type"].(string)
	}

	if _, ok := opt["rmq_key"]; ok {
		self.Key = opt["rmq_key"].(string)
	}

	if _, ok := opt["retries"]; ok {
		retries := opt["retries"].(int64)
		self.Retries = int(retries)
	}
	self.isCheck = true
	self.ExchangeDurable = false
	self.ExchangeAutoDelete = true

	if err := self.InitAmqpClients(); err != nil {
		self.ctx.Logger().Errorln(err)
		self.isCheck = false
	}
}

func (self *RabbitmqOutputService) InitAmqpClients() error {
	var hosts []string
	for _, url := range self.URLs {
		if conn, err := self.getConnection(url); err == nil {
			if ch, err := conn.Channel(); err == nil {
				ch.QueueDeclare(self.Key, true, false, false, false, nil)

				if err != nil {
					return err
				}
				self.amqpClients[url] = amqpClient{
					client:    ch,
					reconnect: make(chan hostpool.HostPoolResponse, 1),
				}
				//重连处理
				go self.reconnect(url)
				hosts = append(hosts, url)
			}
		}

	}
	if len(hosts) == 0 {
		return errors.New("FAIL TO CONNECT AMQP SERVERS")
	}
	self.hostPool = hostpool.New(hosts)
	return nil
}

//获取MQ连接
func (self *RabbitmqOutputService) getConnection(url string) (*amqp.Connection, error) {
	println("get connect from rmq:", url)
	conn, err := amqp.Dial(url)
	return conn, err
}

//重连机制
func (self *RabbitmqOutputService) reconnect(url string) {
	for {
		select {
		case poolResponse := <-self.amqpClients[url].reconnect:
			for {
				time.Sleep(time.Duration(self.ReconnectDelay) * time.Second)
				if conn, err := self.getConnection(poolResponse.Host()); err == nil {
					if ch, err := conn.Channel(); err == nil {

						if err == nil {
							self.amqpClients[poolResponse.Host()] = amqpClient{
								client:    ch,
								reconnect: make(chan hostpool.HostPoolResponse, 1),
							}
							poolResponse.Mark(nil)
							break
						}
					}
				}
				self.ctx.Logger().Infoln("Failed to reconnect to ", url, ". Waiting ", self.ReconnectDelay, " seconds...")
			}
		}
	}
}

func (self *RabbitmqOutputService) Check() bool {
	return self.isCheck
}

func (self *RabbitmqOutputService) DoWrite(packets []*p.Packet) {

	b, err := p.MashallPackets(packets)

	if err != nil {
		return
	}

	for i := 0; i <= self.Retries; i++ {
		hp := self.hostPool.Get()
		if err := self.amqpClients[hp.Host()].client.Publish(
			"",
			self.Key,
			false,
			false,
			amqp.Publishing{
				Body: b,
			},
		); err != nil {
			hp.Mark(err)
			self.amqpClients[hp.Host()].reconnect <- hp
		} else {
			break
		}
	}
}
