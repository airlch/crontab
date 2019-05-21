package master

import (
	"airlch/crontab/common"
	"context"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"log"
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

//保存任务操作
func (JobMgr *JobManage) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	var (
		jobKey   string
		jobValue []byte
		putResp  *clientv3.PutResponse
		jobObj   common.Job
	)

	//etcd 存储 key和val
	jobKey = common.JOB_SAVE_DIR + job.Name
	//任务信息json
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}

	//将数据提交存储到etcd中
	if putResp, err = JobMgr.Kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		log.Println(err)
		return
	}
	//当前一个值存在时，也就是更新
	if putResp.PrevKv != nil {
		//反序列化获取前一个值，报错不影响当前逻辑，要保存的数据已经提交保存到etcd中了
		if err = json.Unmarshal(putResp.PrevKv.Value, &jobObj); err != nil {
			log.Println("SaveJob:etcd获取旧值失败，jobKey：", jobKey)
			err = nil
			return
		}

		oldJob = &jobObj
	}

	return
}

//删除任务操作
func (JobMgr *JobManage) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey      string
		delResp     *clientv3.DeleteResponse
		oldJobBytes []byte
		oldJobObj   common.Job
	)

	//etcd key
	jobKey = common.JOB_SAVE_DIR + name

	//删除操作
	if delResp, err = JobMgr.Kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}

	//判断是否删除成功   删除成功返回删除前的数据
	if delResp.Deleted > 0 {
		oldJobBytes = delResp.PrevKvs[0].Value

		//反序列化   不判断错误信息了，  不影响逻辑
		if err = json.Unmarshal(oldJobBytes, &oldJobObj); err != nil {
			log.Println("DeleteJob:etcd获取旧值失败，jobKey：", jobKey)
			err = nil
			return
		}

		oldJob = &oldJobObj
	}

	return
}

//获取任务列表
func (JobMgr *JobManage) GetJobList() (jobList []*common.Job, err error) {
	var (
		jobPath    string
		getResp    *clientv3.GetResponse
		jobObj     *common.Job
		jobListObj []*common.Job
		kvPair     *mvccpb.KeyValue
	)

	//任务保存目录
	jobPath = common.JOB_SAVE_DIR

	//获取job列表（以jobpath开头）
	if getResp, err = JobMgr.Kv.Get(context.TODO(), jobPath, clientv3.WithPrefix()); err != nil {
		return
	}

	//初始化数组空间
	jobListObj = make([]*common.Job, 0)

	//判断是否有获取到
	if getResp.Count > 0 {
		for _, kvPair = range getResp.Kvs {
			jobObj = &common.Job{}
			//反序列化，
			if err = json.Unmarshal(kvPair.Value, jobObj); err != nil {
				log.Println("GetJobList:etcd获取旧值失败，jobKey：", kvPair.Key)
				continue
			}

			jobListObj = append(jobListObj, jobObj)
		}

		jobList = jobListObj

		err = nil
	}

	return
}

//强杀任务操作
func (JobMgr *JobManage) KillJob(name string) (err error) {
	var (
		jobKey         string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)

	//etcd key
	jobKey = common.JOB_KILLER_DIR + name

	//创建一个租约(过期时间1秒)， put操作让worker监听到后就删除
	if leaseGrantResp, err = JobMgr.Lease.Grant(context.TODO(), 1); err != nil {
		return
	}

	//leaseid
	leaseId = leaseGrantResp.ID

	//put操作，带着1秒租约，让worker监听后到删除  不用拿返回值，只关心是否error就成
	if _, err = JobMgr.Kv.Put(context.TODO(), jobKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return nil
}
