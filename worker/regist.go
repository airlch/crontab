package worker

import (
	"github.com/coreos/etcd/clientv3"
	"net"
	"time"
)

//服务注册
type Regist struct {
	Client  *clientv3.Client
	Kv      clientv3.KV
	Lease   clientv3.Lease
	LocalIP string
}

//单例
var (
	G_Regist *Regist
)

//获取本机网卡地址
func getLocalIP() (ipv4 string, err error) {
	var (
		ipAddrs []net.Addr
		ipAddr  net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)
	// 获取所有网卡
	if ipAddrs, err = net.InterfaceAddrs(); err != nil {
		return
	}

	// 取第一个非lo的网卡IP
	for _, ipAddr = range ipAddrs {
		// 这个网络地址是IP地址: ipv4, ipv6，且不是虚拟网卡
		if ipNet, isIpNet = ipAddr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.String() // 192.168.1.1
				return
			}
		}
	}

	return
}

//服务注册
func (regist *Regist)KeepOnAlive()  {
	
}

//初始化服务注册
func InitRegist() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease

		localIp string
	)

	//初始化配置
	config = clientv3.Config{
		Endpoints:   G_Config.EtcdEndpoints, //集群地址
		DialTimeout: time.Duration(G_Config.EtcdDialTimeout) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	//初始化键值对kv和租约lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	if localIp, err = getLocalIP(); err != nil {
		return
	}

	G_Regist = &Regist{
		Client: client,
		Kv:     kv,
		Lease:  lease,

		LocalIP: localIp,
	}

	//服务注册
	go G_Regist.KeepOnAlive()

	return
}
