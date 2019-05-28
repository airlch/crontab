package worker

import (
	"airlch/crontab/common"
	"context"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"time"
)

//日志保存
type LogSink struct {
	Client        *mongo.Client
	LogCollection *mongo.Collection
	LogChan       chan *common.JobLog
	WorkerChan    chan chan []interface{}
}

//单例
var (
	G_LogSink *LogSink
)

//接收log日志
func (logSink *LogSink) PushLog(log *common.JobLog) {
	logSink.LogChan <- log
}

//日志存储协程
func (logSink *LogSink) LogSinkLoop() {
	var (
		logInfo *common.JobLog
		worker  chan []interface{}

		logInfoQueue []interface{}
		workerQueue  []chan []interface{}

		timeTicker *time.Ticker //定时器
	)

	//定时器   每秒执行1次
	timeTicker = time.NewTicker(1 * time.Second)

	for {

		var (
			tempLogInfoBatch   []interface{}
			tempWorker         chan []interface{}
			logInfoQueueLength int
		)

		//工作worker存在且数据量到100条做存储
		if len(logInfoQueue) > 100 && len(workerQueue) > 0 {
			logInfoQueueLength = len(logInfoQueue)
			tempLogInfoBatch = logInfoQueue[:logInfoQueueLength]
			tempWorker = workerQueue[0]
		}
		select {
		case logInfo = <-logSink.LogChan:
			logInfoQueue = append(logInfoQueue, logInfo)
		case worker = <-logSink.WorkerChan:
			workerQueue = append(workerQueue, worker)
		case tempWorker <- tempLogInfoBatch:
			workerQueue = workerQueue[1:]
			logInfoQueue = logInfoQueue[0:0]
		case <-timeTicker.C: //每秒执行一次
			if len(logInfoQueue) > 0 && len(workerQueue) > 0 {
				logInfoQueueLength = len(logInfoQueue)
				tempLogInfoBatch = logInfoQueue[:logInfoQueueLength]
				tempWorker = workerQueue[0]

				//这里是有可能堵塞的
				tempWorker <- tempLogInfoBatch

				workerQueue = workerQueue[1:]
				logInfoQueue = logInfoQueue[0:0]
			}
		}
	}
}

//存储操作	批量操作，缓解数据库压力
func (logSink *LogSink) SaveLogBatch(logBatch []interface{}) {
	logSink.LogCollection.InsertMany(context.TODO(), logBatch)
}

//创建存储操作工作协程（多个协程抢数据，防止数据量过大，chan超过队列长度死锁数据丢失）
func (logSink *LogSink) CreateLogSinkWorker() {
	var (
		in           chan []interface{}
		logInfoBatch []interface{}
	)

	in = make(chan []interface{})

	for {
		//在工作队列中排队
		logSink.WorkerChan <- in

		//接收数据
		logInfoBatch = <-in

		//批量存储
		logSink.SaveLogBatch(logInfoBatch)
	}
}

//初始化日志保存
func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(
		context.TODO(),
		G_Config.MongodbUri,
		clientopt.ConnectTimeout(time.Duration(G_Config.MongodbTimeOut)*time.Millisecond)); err != nil {
		return
	}

	G_LogSink = &LogSink{
		Client:        client,
		LogCollection: client.Database("cron").Collection("log"),
		LogChan:       make(chan *common.JobLog),
		WorkerChan: make(chan chan []interface{}),
	}

	//启动日志存储协程
	go G_LogSink.LogSinkLoop()

	for i := 0; i < G_Config.LogSinkWorkCount; i++ {
		go G_LogSink.CreateLogSinkWorker()
	}

	return
}
