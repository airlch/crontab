package worker

import (
	"airlch/crontab/common"
	"context"
	"github.com/coreos/etcd/clientv3"
	"log"
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
				ipv4 = ipNet.IP.String() // 192.168.1.1
				return
			}
		}
	}

	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

//服务注册
func (regist *Regist) KeepOnAlive() {
	var (
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
		err            error
		keepAliveChan  <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResp  *clientv3.LeaseKeepAliveResponse
		regKey         string
	)

	for {
		// 注册路径
		regKey = common.JOB_WORKER_DIR + regist.LocalIP

		cancelFunc = nil

		//创建租约
		if leaseGrantResp, err = regist.Lease.Grant(context.TODO(), 5); err != nil {
			goto RETRY
		}

		//获取租约id
		leaseId = leaseGrantResp.ID

		cancelCtx, cancelFunc = context.WithCancel(context.TODO())

		//续租
		if keepAliveChan, err = regist.Lease.KeepAlive(cancelCtx, leaseId); err != nil {
			goto RETRY
		}

		if _, err = regist.Kv.Put(context.Background(), regKey, "", clientv3.WithLease(leaseId)); err != nil {
			goto RETRY
		}


		//处理续租应答
		for {
			select {
			case keepAliveResp = <-keepAliveChan:
				if keepAliveResp == nil { //续租失败
					goto RETRY
				}
			}
		}

	RETRY:
		time.Sleep(1 * time.Second)
		if cancelFunc != nil {
			cancelFunc()
			regist.Lease.Revoke(context.TODO(), leaseId)
		}
	}

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
		log.Println("获取本地网卡失败！")
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
