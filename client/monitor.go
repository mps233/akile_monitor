package main

import (
	"akile_monitor/client/model"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
)

func GetState() *model.HostState {
	var ret model.HostState
	cp, err := cpu.Percent(0, false)
	if err != nil || len(cp) == 0 {
		log.Println("cpu.Percent error:", err)
	} else {
		ret.CPU = cp[0]
	}

	loadStat, err := load.Avg()
	if err != nil {
		log.Println("load.Avg error:", err)
	} else {
		ret.Load1 = Decimal(loadStat.Load1)
		ret.Load5 = Decimal(loadStat.Load5)
		ret.Load15 = Decimal(loadStat.Load15)

	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("mem.VirtualMemory error:", err)
	} else {
		ret.MemUsed = vm.Total - vm.Available
	}

	uptime, err := host.Uptime()
	if err != nil {
		log.Println("host.Uptime error:", err)
	} else {
		ret.Uptime = uptime
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		log.Println("mem.SwapMemory error:", err)
	} else {
		ret.SwapUsed = swap.Used
	}

	ret.NetInTransfer, ret.NetOutTransfer = netInTransfer, netOutTransfer
	ret.NetInSpeed, ret.NetOutSpeed = netInSpeed, netOutSpeed

	return &ret

}

func GetHost() *model.Host {
	var ret model.Host
	ret.Name = cfg.Name
	var cpuType string
	hi, err := host.Info()
	if err != nil {
		log.Println("host.Info error:", err)
	}
	cpuType = "Virtual"
	ret.Platform = hi.Platform
	ret.PlatformVersion = hi.PlatformVersion
	ret.Arch = hi.KernelArch
	ret.Virtualization = hi.VirtualizationSystem
	ret.BootTime = hi.BootTime
	ci, err := cpu.Info()
	if err != nil {
		log.Println("cpu.Info error:", err)
	}
	ret.CPU = append(ret.CPU, fmt.Sprintf("%s %d %s Core", ci[0].ModelName, runtime.NumCPU(), cpuType))
	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Println("mem.VirtualMemory error:", err)
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		log.Println("mem.SwapMemory error:", err)
	}

	ret.MemTotal = vm.Total
	ret.SwapTotal = swap.Total

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println("获取网络接口失败:", err)
	} else {
		log.Printf("找到 %d 个网络接口", len(interfaces))

		for _, i := range interfaces {
			log.Printf("检查网络接口: %s", i.Name)

			addrs, err := i.Addrs()
			if err != nil {
				log.Printf("获取接口 %s 的地址失败: %v", i.Name, err)
				continue
			}

			log.Printf("接口 %s 有 %d 个地址", i.Name, len(addrs))

			for _, addr := range addrs {
				log.Printf("处理地址: %v", addr.String())

				if ipnet, ok := addr.(*net.IPNet); ok {
					if ipnet.IP.IsLoopback() {
						log.Printf("跳过回环地址: %v", ipnet.IP)
						continue
					}

					if ipnet.IP.To4() != nil {
						log.Printf("找到有效的IPv4地址: %v", ipnet.IP)
						ret.IP = append(ret.IP, ipnet.IP.String())
					} else {
						log.Printf("跳过非IPv4地址: %v", ipnet.IP)
					}
				} else {
					log.Printf("地址类型转换失败: %T", addr)
				}
			}
		}
	}

	if len(ret.IP) == 0 {
		log.Println("警告: 未找到任何有效的IP地址")
	} else {
		log.Printf("最终获取到的IP地址列表: %v", ret.IP)
	}

	return &ret

}

var (
	netInSpeed, netOutSpeed, netInTransfer, netOutTransfer, lastUpdateNetStats uint64
)

// TrackNetworkSpeed NIC监控，统计流量与速度
func TrackNetworkSpeed() {
	var innerNetInTransfer, innerNetOutTransfer uint64
	nc, err := psnet.IOCounters(true)
	if err == nil {
		for _, v := range nc {
			if v.Name == cfg.NetName {
				innerNetInTransfer += v.BytesRecv
				innerNetOutTransfer += v.BytesSent
			}
		}
		now := uint64(time.Now().Unix())
		diff := now - lastUpdateNetStats
		if diff > 0 {
			netInSpeed = (innerNetInTransfer - netInTransfer) / diff
			netOutSpeed = (innerNetOutTransfer - netOutTransfer) / diff
		}
		netInTransfer = innerNetInTransfer
		netOutTransfer = innerNetOutTransfer
		lastUpdateNetStats = now

	}
}

// 保留两位小数
func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}
