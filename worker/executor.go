package worker

import (
	"airlch/crontab/common"
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
	ExecuteInfoChan   chan *common.JobExecuteInfo
	ExecuteWorkerChan chan chan *common.JobExecuteInfo
}

//单例
var (
	G_Executor *Executor
)

//推送执行任务
func (executor *Executor) PushExecuteJob(info *common.JobExecuteInfo) {
	executor.ExecuteInfoChan <- info
}

//监听并执行任务
func (executor *Executor) ExecuteLoop() {
	var (
		execInfo   *common.JobExecuteInfo
		execWorker chan *common.JobExecuteInfo

		//任务信息和执行任务协程队列
		execInfoQueue   []*common.JobExecuteInfo
		execWorkerQueue []chan *common.JobExecuteInfo
	)

	for {
		//chan不实例化，select不会接收到值
		var tempExecInfo *common.JobExecuteInfo
		var tempExecWorker chan *common.JobExecuteInfo

		//当任务信息和任务工作协程都存在的时候，把任务信息调度给协程，执行任务
		if len(execInfoQueue) > 0 && len(execWorkerQueue) > 0 {
			tempExecInfo = execInfoQueue[0]
			tempExecWorker = execWorkerQueue[0]
		}

		select {
		case execInfo = <-executor.ExecuteInfoChan: //任务信息放进队列
			execInfoQueue = append(execInfoQueue, execInfo)
		case execWorker = <-executor.ExecuteWorkerChan: //任务工作协程放进队列
			execWorkerQueue = append(execWorkerQueue, execWorker)
		case tempExecWorker <- tempExecInfo:
			execInfoQueue = execInfoQueue[1:]
			execWorkerQueue = execWorkerQueue[1:]
		}
	}
}

//创建执行工作协程
func (executor *Executor) CreateExecuteWorker() {
	var (
		in   chan *common.JobExecuteInfo
		info *common.JobExecuteInfo
	)

	in = make(chan *common.JobExecuteInfo)

	for {
		//初始化把（N个）工作协程丢进execute工作队列里
		executor.ExecuteWorkerChan <- in

		//等待，排队领取任务信息，抢到执行任务
		info = <-in

		executor.HandleExecute(info)
	}
}

//执行任务
func (executor *Executor) HandleExecute(info *common.JobExecuteInfo) {
	var (
		cmd           *exec.Cmd
		output        []byte
		err           error
		executeResult *common.JobExecuteResult
		jobLock       *JobLock
	)

	//todo:执行任务

	//构造执行任务结果
	executeResult = &common.JobExecuteResult{
		ExecuteInfo: info,
	}

	//首先获取分布式锁
	jobLock = G_JobManage.CreateJobLock(info.Job.Name)

	//任务执行开始时间
	executeResult.StartTime = time.Now()

	//随机睡眠（0-1s）
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	//尝试上锁
	err = jobLock.TryLock()
	defer jobLock.Unlock() //执行完释放锁

	if err != nil {
		executeResult.Err = err
		executeResult.EndTime = time.Now()
	} else {
		executeResult.StartTime = time.Now() //开始时间重新计算

		//执行shell命令		//可取消的context，强杀时调用其取消函数
		cmd = exec.CommandContext(info.CancelCtx, "e:\\soft\\cygwin\\bin\\bash.exe", "-c", info.Job.Command)

		//执行并捕获输出
		output, err = cmd.CombinedOutput()

		//任务执行结束时间
		executeResult.EndTime = time.Now()
		//任务输出和错误信息
		executeResult.Output = output
		executeResult.Err = err

		time.Sleep(400 * time.Millisecond)
	}

	fmt.Println("执行任务：", info.Job.Name, info.PlanTime, info.RealTime, string(output), err)

	G_Schedule.PushExecuteResult(executeResult)

}

//初始化执行器
func InitExecutor() (err error) {
	G_Executor = &Executor{
		ExecuteInfoChan:   make(chan *common.JobExecuteInfo),
		ExecuteWorkerChan: make(chan chan *common.JobExecuteInfo),
	}

	//监听任务执行
	go G_Executor.ExecuteLoop()

	//配置执行器数量
	for i := 0; i < G_Config.ExecuteWorkCount; i++ {
		go G_Executor.CreateExecuteWorker()
	}

	return
}
