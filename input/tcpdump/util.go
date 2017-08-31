package tcpdump

import (
	"encoding/json"
	"errors"
	"fmt"
	util "github.com/domac/mafio/util"
	"github.com/google/gopacket/pcap"
	"github.com/shirou/gopsutil/net"
	"reflect"
	"strconv"
	"strings"
)

func isLoopback(device pcap.Interface) bool {
	if len(device.Addresses) == 0 {
		return false
	}

	switch device.Addresses[0].IP.String() {
	case "127.0.0.1", "::1":
		return true
	}

	return false
}

//获取要监听的网卡
func findDevices() []string {
	devices, err := pcap.FindAllDevs()

	if err != nil {
		return []string{}
	}

	interfaces := []string{}
	for _, device := range devices {

		//不处理没绑定地址的网卡
		if len(device.Addresses) == 0 || isLoopback(device) {
			continue
		}

		if strings.HasPrefix(device.Name, "lo") {
			continue
		}

		//如果是绑定的网卡,立刻返回
		if strings.HasPrefix(device.Name, "bond") {
			return []string{device.Name}
		}
		interfaces = append(interfaces, device.Name)
	}
	return interfaces

}

//获取配置
func convertConfig(arg interface{}) (out []string, ok bool) {
	//类型转换
	slice, success := convertArg(arg, reflect.Slice)
	if !success {
		ok = false
		return
	}

	c := slice.Len()
	out = make([]string, c)
	for i := 0; i < c; i++ {
		tmp := slice.Index(i).Interface()
		if tmp != nil {
			//强制转换为字符串
			out[i] = tmp.(string)
		}
	}

	return out, true
}

func convertArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

func getPidByName(name string) (pids []int32) {

	cmd := fmt.Sprintf("ps aux | grep -i %s |grep -v 'grep' | awk '{print $2}'", name)
	result, err := util.ShellString(cmd)
	if err != nil {
		return
	}
	resultList := strings.Split(result, "\n")
	for _, r := range resultList {
		if r == "" || r == "0" || r == "1" {
			continue
		}

		pid, err := strconv.Atoi(r)
		if err != nil {
			continue
		}
		pids = append(pids, int32(pid))
	}
	return
}

//根据进程号获取所属的端口
func getPortsByProcessId(pid int32) (ports []string) {
	if pid == 0 {
		return
	}

	niters, err := net.ConnectionsPid("all", pid)
	if err != nil {
		return
	}
	for _, ns := range niters {
		lport := strconv.Itoa(int(ns.Laddr.Port))
		ports = append(ports, lport)
	}
	return
}

func getPortsByProcessName(name string) (ports []string, err error) {
	if name == "" {
		err = errors.New("process name is null")
		return
	}

	pids := getPidByName(name)
	if pids == nil {
		err = errors.New("pids is null")
		return
	}

	pmap := make(map[string]string)

	for _, pid := range pids {
		processPorts := getPortsByProcessId(pid)
		if processPorts == nil {
			continue
		}
		for _, p := range processPorts {
			if p == "0" {
				continue
			}

			if _, ok := pmap[p]; ok {
				continue
			}

			//排重
			pmap[p] = p
			ports = append(ports, p)
		}
	}
	return
}

func JsonStringToMap(jstr string) (map[string]interface{}, error) {
	resultMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(jstr), &resultMap); err != nil {
		return nil, err
	}
	return resultMap, nil
}
