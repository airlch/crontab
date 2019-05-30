package master

import (
	"airlch/crontab/common"
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"time"
)

//服务发现
type WorkerManage struct {
	Client *clientv3.Client
	Kv     clientv3.KV
	Lease  clientv3.Lease
	Watch  clientv3.Watcher

	WorkerMap map[string]string	//worker列表，本地内存保存，与etcd同步
}

//单例
var (
	G_WorkerManage *WorkerManage
)

//获取健康节点，服务发现
//func (workerMgr *WorkerManage) WorkerList() (workerIpArr []string, err error) {
//	var (
//		getResp  *clientv3.GetResponse
//		kvItem   *mvccpb.KeyValue
//		workerIp string
//	)
//
//	if getResp, err = workerMgr.Kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix()); err != nil {
//		return
//	}
//
//	for _, kvItem = range getResp.Kvs {
//		workerIp = common.ExtractWorkerName(string(kvItem.Key))
//
//		workerIpArr = append(workerIpArr, workerIp)
//	}
//
//	return
//}

//获取健康节点，服务发现
func (workerMgr *WorkerManage) WorkerList() (workerIpArr []string, err error) {
	var (
		worker string
	)

	for worker = range workerMgr.WorkerMap {
		workerIpArr = append(workerIpArr, worker)
	}

	return
}

//监听健康节点
func (workerMgr *WorkerManage) WatchWorker() (err error) {
	var (
		getResp    *clientv3.GetResponse
		kvItem     *mvccpb.KeyValue
		workerIP   string
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
	)

	//获取workerip数据
	if getResp, err = workerMgr.Kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	//本地内存保存
	for _, kvItem = range getResp.Kvs {
		workerIP = common.ExtractWorkerName(string(kvItem.Key))

		workerMgr.WorkerMap[workerIP] = workerIP
	}

	watchChan = workerMgr.Watch.Watch(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix())

	//同步本地内存
	for watchResp = range watchChan {
		for _, watchEvent = range watchResp.Events {
			workerIP = common.ExtractWorkerName(string(watchEvent.Kv.Key))
			switch watchEvent.Type {
			case mvccpb.PUT:
				workerMgr.WorkerMap[workerIP] = workerIP
			case mvccpb.DELETE:
				delete(workerMgr.WorkerMap, workerIP)
			}
		}
	}

	return
}

//初始化服务发现
func InitWorkerManage() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
		watch  clientv3.Watcher
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
	watch = clientv3.NewWatcher(client)

	G_WorkerManage = &WorkerManage{
		Client: client,
		Kv:     kv,
		Lease:  lease,
		Watch:  watch,

		WorkerMap: make(map[string]string),
	}

	//监听健康节点，同步本地内存
	go G_WorkerManage.WatchWorker()

	return
}
