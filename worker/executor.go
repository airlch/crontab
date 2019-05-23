package worker

import (
	"airlch/crontab/common"
	"context"
	"fmt"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
	ExecuteChan chan *common.JobExecuteInfo
}

//单例
var (
	G_Executor *Executor
)

//推送执行任务
func (executor *Executor) PushExecuteJob(info *common.JobExecuteInfo) {
	executor.ExecuteChan <- info
}

//监听并执行任务
func (executor *Executor) ExecuteLoop() {
	var (
		execInfo *common.JobExecuteInfo
	)

	for {
		select {
		case execInfo = <-executor.ExecuteChan:
			executor.HandleExecute(execInfo)
		}
	}
}

//执行任务
func (executor *Executor) HandleExecute(info *common.JobExecuteInfo) {
	var (
		cmd           *exec.Cmd
		output        []byte
		err           error
		executeResult *common.JobExecuteResult
	)

	//todo:执行任务

	//构造执行任务结果
	executeResult = &common.JobExecuteResult{
		ExecuteInfo: info,
	}

	//任务执行开始时间
	executeResult.StartTime = time.Now()

	//执行shell命令
	cmd = exec.CommandContext(context.TODO(), "e:\\soft\\cygwin\\bin\\bash.exe", "-c", info.Job.Command)

	//执行并捕获输出
	output, err = cmd.CombinedOutput()

	//任务执行结束时间
	executeResult.EndTime = time.Now()
	//任务输出和错误信息
	executeResult.Output = output
	executeResult.Err = err

	fmt.Println("执行任务：", info.Job.Name, info.PlanTime, info.RealTime, string(output), err)

	G_Schedule.PushExecuteResult(executeResult)
}

//初始化执行器
func InitExecutor() (err error) {
	G_Executor = &Executor{
		ExecuteChan: make(chan *common.JobExecuteInfo, 1000),
	}

	//监听任务执行
	go G_Executor.ExecuteLoop()

	return
}
