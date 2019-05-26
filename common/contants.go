package common

const (
	//任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	//强杀任务目录
	JOB_KILLER_DIR = "/cron/kill/"

	//变化事件更新操作
	JOB_EVENT_SAVE_TYPE = 1

	//变化事件删除操作
	JOB_EVENT_DELETE_TYPE = 2

	//分布式锁路径
	JOB_LOCK_DIR = "/cron/lock/"
)
