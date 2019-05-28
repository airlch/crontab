package worker

import (
	"airlch/crontab/common"
	"fmt"
	"time"
)

//任务调度
type Schedule struct {
	JobEventChan         chan *common.JobEvent              //etcd任务事件队列
	JobSchedulePlanTable map[string]*common.JobSchedulePlan //任务调度计划表
	JobExecuteInfoTable  map[string]*common.JobExecuteInfo  //任务执行状态表
	JobExecuteResultChan chan *common.JobExecuteResult      //任务执行结果队列
}

//单例
var (
	G_Schedule *Schedule
)

//推送调度任务
func (schedule *Schedule) PushJobEvent(jobEvent *common.JobEvent) {
	schedule.JobEventChan <- jobEvent
}

//推送任务执行结果
func (schedule *Schedule) PushExecuteResult(info *common.JobExecuteResult) {
	schedule.JobExecuteResultChan <- info
}

func (schedule *Schedule) HandleExecuteResult(info *common.JobExecuteResult) {
	var (
		jobExecuteInfoIsExist bool
		jobLog                *common.JobLog
	)

	//同步内存，执行完成任务
	if _, jobExecuteInfoIsExist = G_Schedule.JobExecuteInfoTable[info.ExecuteInfo.Job.Name]; jobExecuteInfoIsExist {
		delete(G_Schedule.JobExecuteInfoTable, info.ExecuteInfo.Job.Name)
	}

	//实例化日志数据插入类
	jobLog = &common.JobLog{
		JobName:      info.ExecuteInfo.Job.Name,
		Command:      info.ExecuteInfo.Job.Command,
		Output:       string(info.Output),
		PlanTime:     info.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
		ScheduleTime: info.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
		StartTime:    info.StartTime.UnixNano() / 1000 / 1000,
		EndTime:      info.EndTime.UnixNano() / 1000 / 1000,
	}

	if info.Err != nil {
		jobLog.Err = info.Err.Error()
	} else {
		jobLog.Err = ""
	}

	//TODO：插入mongodb
	G_LogSink.PushLog(jobLog)

	fmt.Println("任务执行完成", info.ExecuteInfo.Job.Name, string(info.Output), info.Err)
}

//尝试启动任务
func (schedule *Schedule) TryStartJob(jobPlan *common.JobSchedulePlan) {
	//调度和执行是2件事情

	//一个1秒调度1次的任务，任务可能执行很久（1分钟），这时候1分钟会调度60次，但是只能执行1次   去重 防止并发

	var (
		jobExecuteInfo        *common.JobExecuteInfo
		jobExecuteInfoIsExist bool
	)

	//如果任务正在执行，跳过本次调度
	if jobExecuteInfo, jobExecuteInfoIsExist = schedule.JobExecuteInfoTable[jobPlan.Job.Name]; jobExecuteInfoIsExist {
		fmt.Println("尚未退出，跳过执行：", jobExecuteInfo.Job.Name)
		return
	}

	//构造执行任务状态
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	//保存执行状态
	schedule.JobExecuteInfoTable[jobPlan.Job.Name] = jobExecuteInfo

	//todo:执行任务
	G_Executor.PushExecuteJob(jobExecuteInfo)
	//fmt.Println("执行任务：", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
}

//重新计算任务调度状态
func (schedule *Schedule) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan      *common.JobSchedulePlan
		now          time.Time
		nextPlanTime *time.Time
	)

	now = time.Now()

	//不存在调度任务，睡眠1秒
	if len(schedule.JobSchedulePlanTable) == 0 {
		scheduleAfter = time.Second * 1
		return
	}

	//1.遍历所有任务

	//2.过期的任务立即执行

	//3.统计最近要过期的任务（N秒后过期==scheduleAfter）

	for _, jobPlan = range schedule.JobSchedulePlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//todo:执行任务
			schedule.TryStartJob(jobPlan)

			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}

		if nextPlanTime == nil || jobPlan.NextTime.Before(*nextPlanTime) {
			nextPlanTime = &jobPlan.NextTime
		}
	}

	scheduleAfter = (*nextPlanTime).Sub(now)

	return scheduleAfter
}

//处理任务事件
func (schedule *Schedule) HandleJobEvent(jobEvent *common.JobEvent) (err error) {
	var (
		jobSchedulePlan    *common.JobSchedulePlan
		jobScheduleIsExist bool
		jobExecuteInfo     *common.JobExecuteInfo
		jobExecuteIsExist  bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE_TYPE: //保存任务事件
		//解析任务，构建任务调度计划
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}

		//同步更改到内存中的调度计划表中，   保持etcd和内存一致
		schedule.JobSchedulePlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE_TYPE: //删除任务事件
		//判断内存中是否存在，etcd中不存在了也是会接收到删除事件	保持etcd和内存一致
		if jobSchedulePlan, jobScheduleIsExist = schedule.JobSchedulePlanTable[jobEvent.Job.Name]; jobScheduleIsExist {
			delete(schedule.JobSchedulePlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL_TYPE: //强杀任务事件
		//判断是否存在执行中的任务，存在则执行取消函数
		if jobExecuteInfo, jobExecuteIsExist = schedule.JobExecuteInfoTable[jobEvent.Job.Name]; jobExecuteIsExist {
			jobExecuteInfo.CancelFunc()
		}
	}

	return
}

//监听并执行调度变化任务
func (schedule *Schedule) ScheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTime  *time.Timer
		err           error
		executeResult *common.JobExecuteResult
	)

	//初始化计算调度任务，睡眠1秒
	scheduleAfter = schedule.TrySchedule()
	scheduleTime = time.NewTimer(scheduleAfter)

	for {
		//判断是否重新计算调度服务，只有监听到任务变化或者任务到期需要重新计算，接收任务结果不需要操作
		var isScheduleExecute = false

		select {
		case jobEvent = <-schedule.JobEventChan: //监听任务变化事件
			isScheduleExecute = true
			//处理任务事件
			if err = schedule.HandleJobEvent(jobEvent); err != nil {
				continue
			}
		case <-scheduleTime.C: //最近的调度任务到期了，需要执行调度
			isScheduleExecute = true
		case executeResult = <-schedule.JobExecuteResultChan: //接收任务执行结果
			schedule.HandleExecuteResult(executeResult)
		}

		if isScheduleExecute {
			//当监听到事件变化或者调度任务过期，重新计算调度任务，执行
			scheduleAfter = schedule.TrySchedule()
			//重新计算调度间隔
			scheduleTime = time.NewTimer(scheduleAfter)
		}
	}
}

//初始化调度器
func InitSchedule() (err error) {
	G_Schedule = &Schedule{
		JobEventChan:         make(chan *common.JobEvent, 1000),
		JobSchedulePlanTable: make(map[string]*common.JobSchedulePlan),
		JobExecuteInfoTable:  make(map[string]*common.JobExecuteInfo),
		JobExecuteResultChan: make(chan *common.JobExecuteResult, 1000),
	}

	//启动调度协程
	go G_Schedule.ScheduleLoop()

	return
}
