package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//任务模型
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
}

//变化事件
type JobEvent struct {
	EventType int64 //1，更新 2，删除
	Job       *Job
}

//任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression //解析好的cronexpr表达式
	NextTime time.Time            //下次调度时间
}

//任务执行状态
type JobExecuteInfo struct {
	Job        *Job               //任务信息
	PlanTime   time.Time          //理论调度时间
	RealTime   time.Time          //真实调度时间
	CancelCtx  context.Context    //可取消的context
	CancelFunc context.CancelFunc //取消函数
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行状态
	Output      []byte          //脚本输出
	Err         error           //脚本错误
	StartTime   time.Time       //任务开始时间
	EndTime     time.Time       //任务结束时间
}

// 任务执行日志
type JobLog struct {
	JobName string `json:"jobName" bson:"jobName"` // 任务名字
	Command string `json:"command" bson:"command"` // 脚本命令
	Err string `json:"err" bson:"err"` // 错误原因
	Output string `json:"output" bson:"output"`	// 脚本输出
	PlanTime int64 `json:"planTime" bson:"planTime"` // 计划开始时间
	ScheduleTime int64 `json:"scheduleTime" bson:"scheduleTime"` // 实际调度时间
	StartTime int64 `json:"startTime" bson:"startTime"` // 任务执行开始时间
	EndTime int64 `json:"endTime" bson:"endTime"` // 任务执行结束时间
}

//HTTP接口应答
type Response struct {
	ErrNo   int         `json:"errNo"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

//应答方法
func BuildResponse(errNo int, message string, data interface{}) (respInfo []byte, err error) {
	//1.定义一个response
	var (
		response Response
	)

	response = Response{
		ErrNo:   errNo,
		Message: message,
		Data:    data,
	}

	//2.序列化json
	respInfo, err = json.Marshal(response)

	return
}

//反序列化job
func UnmarshalJob(byte []byte) (job *Job, err error) {
	var (
		jobObj *Job
	)

	jobObj = &Job{}

	if err = json.Unmarshal(byte, jobObj); err != nil {
		return
	}

	job = jobObj

	return
}

//从etcd的key中提取任务名
// /cron/job/job1 抹掉 /cron/job/
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

// /cron/kill/job1 获得job1
func ExtractKillName(killKey string) string {
	return strings.TrimPrefix(killKey, JOB_KILLER_DIR)
}

// /cron/workers/job1 获得job1
func ExtractWorkerName(workerKey string) string {
	return strings.TrimPrefix(workerKey, JOB_WORKER_DIR)
}

//构造任务变化事件
//任务变化事件有两种    1，更新  2，删除
func BuildJobEvent(eventType int64, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

//构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr               *cronexpr.Expression
		jobSchedulePlanObj *JobSchedulePlan
	)

	//解析cron表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}

	jobSchedulePlanObj = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	jobSchedulePlan = jobSchedulePlanObj

	return
}

//构造任务执行状态信息
func BuildJobExecuteInfo(jobPlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobPlan.Job,
		PlanTime: jobPlan.NextTime, //计算调度时间
		RealTime: time.Now(),       //真实调度时间
	}

	//带取消的context和取消函数
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())

	return
}
