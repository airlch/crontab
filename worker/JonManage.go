package worker

import (
	"github.com/coreos/etcd/clientv3"
	"time"
)

//任务管理器
type JobManage struct {
	Client *clientv3.Client
	Kv     clientv3.KV
	Lease  clientv3.Lease
}

//单例对象
var (
	G_JobManage *JobManage
)

//初始化任务管理器
func InitJobManage() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	//赋值单例
	G_JobManage = &JobManage{
		Client: client,
		Kv:     kv,
		Lease:  lease,
	}

	return
}
