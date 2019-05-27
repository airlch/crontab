package worker

import (
	"airlch/crontab/common"
	"context"
	"github.com/coreos/etcd/clientv3"
)

//分布式锁(TXN事务)
type JobLock struct {
	Kv    clientv3.KV
	Lease clientv3.Lease

	JobName string //任务名称
	//ContextCtx context.Context
	ContextFunc context.CancelFunc //取消方法
	leaseId     clientv3.LeaseID   //租约id
	IsLock      bool               //是否上锁
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		Kv:      kv,
		Lease:   lease,
		JobName: jobName,
		IsLock:  false,
	}

	return
}

//尝试上锁
func (jobLockObj *JobLock) TryLock() (err error) {
	var (
		leaseResp         *clientv3.LeaseGrantResponse
		contextCtx        context.Context
		contextFunc       context.CancelFunc
		leaseId           clientv3.LeaseID
		keepAliveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		txn               clientv3.Txn
		lockKey           string
		txnResp           *clientv3.TxnResponse
	)

	//1.创建租约（3秒）
	if leaseResp, err = jobLockObj.Lease.Grant(context.TODO(), 3); err != nil {
		return
	}

	//租约id
	leaseId = leaseResp.ID

	//context用于取消自动续租
	contextCtx, contextFunc = context.WithCancel(context.TODO())

	//2.自动续租
	if keepAliveRespChan, err = jobLockObj.Lease.KeepAlive(contextCtx, leaseId); err != nil {
		goto FAIL
	}

	//3.处理续租应答的协程
	go func() {
		var (
			keepAliveResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepAliveResp = <-keepAliveRespChan: //自动续租应答，心跳
				if keepAliveResp == nil { //判断自动续租是否取消
					goto END
				}
			}
		}
	END:
	}()

	//4.创建事务txn
	txn = jobLockObj.Kv.Txn(context.TODO())

	//锁路径
	lockKey = common.JOB_LOCK_DIR + jobLockObj.JobName

	//5.事务抢锁
	//txn抢分布式锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))

	//判断是否提交成功，未返回成功的都直接放弃掉租约
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	//6.成功返回，失败回滚
	if !txnResp.Succeeded {
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	jobLockObj.ContextFunc = contextFunc
	jobLockObj.leaseId = leaseId
	jobLockObj.IsLock = true

	return

FAIL:
	contextFunc()                                    //取消自动续租
	jobLockObj.Lease.Revoke(context.TODO(), leaseId) //立即释放租约
	return
}

//释放锁
func (jobLockObj *JobLock) Unlock() {
	if jobLockObj.IsLock {
		jobLockObj.ContextFunc()
		jobLockObj.Lease.Revoke(context.TODO(), jobLockObj.leaseId)
	}
}
