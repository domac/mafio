package agent

import (
	m "github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"time"
)

func DoMetrics(influxdb_addr, influxdb_db, user, password string, interval time.Duration) {
	//debug输出信息(输出测试)
	r := m.NewRegistry()
	m.RegisterDebugGCStats(r)
	m.RegisterRuntimeMemStats(r)
	go m.CaptureDebugGCStats(r, interval)
	go m.CaptureRuntimeMemStats(r, interval)
	go influxdb.InfluxDB(
		r,             // metrics registry
		interval,      // 时间间隔
		influxdb_addr, // InfluxDB url
		influxdb_db,   // InfluxDB 数据库名
		user,          // InfluxDB user
		password,      // InfluxDB password
	)
}
