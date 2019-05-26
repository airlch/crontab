package worker

import (
	"airlch/crontab/common"
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"time"
)

//任务管理器
type JobManage struct {
	Client  *clientv3.Client
	Kv      clientv3.KV
	Lease   clientv3.Lease
	Watcher clientv3.Watcher
}

//单例对象
var (
	G_JobManage *JobManage
)

//监听任务变化
func (jobMgr *JobManage) WatchJobs() (err error) {
	var (
		getResp       *clientv3.GetResponse
		keyPair       *mvccpb.KeyValue
		watchRevision int64
		watchChan     clientv3.WatchChan
		watchResp     clientv3.WatchResponse
		job           *common.Job
		event         *clientv3.Event
		jobName       string
		jobEvent      *common.JobEvent
	)

	//获取目录下的所有任务
	if getResp, err = jobMgr.Kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	//当前有哪些任务
	for _, keyPair = range getResp.Kvs {
		//反序列化json得到job
		if job, err = common.UnmarshalJob(keyPair.Value); err != nil {
			err = nil
			continue
		}

		//jobEvent = &common.JobEvent{EventType: common.JOB_EVENT_SAVE_TYPE, Job: job}
		jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE_TYPE, job)

		//todo:之后可以调度给schedule(调度协程)
		G_Schedule.PushJobEvent(jobEvent)

		fmt.Println(*jobEvent)
	}

	//监听协程
	go func() {
		//监听get时刻的后续版本变化
		watchRevision = getResp.Header.Revision + 1

		//监听/cron/jobs目录下的后续变化
		watchChan = jobMgr.Watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchRevision), clientv3.WithPrefix())

		//监听
		for watchResp = range watchChan {
			for _, event = range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT: //任务保存事件
					if job, err = common.UnmarshalJob(event.Kv.Value); err != nil {
						err = nil
						continue
					}

					//构建一个更新event事件
					//jobEvent = &common.JobEvent{EventType: common.JOB_EVENT_SAVE_TYPE, Job: job}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE_TYPE, job)

				case mvccpb.DELETE: //任务删除事件
					//delete /cron/job/job1    需要提取出job1
					jobName = common.ExtractJobName(string(event.Kv.Key))

					//构建一个删除event事件
					//jobEvent = &common.JobEvent{EventType: common.JOB_EVENT_DELETE_TYPE, Job: job}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE_TYPE, job)

				}

				//todo:推送一个事件给schedule(删除，更新)
				G_Schedule.PushJobEvent(jobEvent)

				fmt.Println(*jobEvent)
			}
		}
	}()

	return
}

//初始化任务管理器
func InitJobManage() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
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
	watcher = clientv3.NewWatcher(client)

	//赋值单例
	G_JobManage = &JobManage{
		Client:  client,
		Kv:      kv,
		Lease:   lease,
		Watcher: watcher,
	}

	// 启动任务监听
	G_JobManage.WatchJobs()

	return
}

//创建任务执行锁
func (jobMgr *JobManage) CreateJobLock(jobName string) (JobLock *JobLock) {
	//返回一把锁
	JobLock = InitJobLock(jobName, jobMgr.Kv, jobMgr.Lease)

	return
}
