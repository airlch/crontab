package master

import (
	"airlch/crontab/common"
	"context"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"log"
	"time"
)

//日志管理
type LogManage struct {
	Client        *mongo.Client
	LogCollection *mongo.Collection
}

//过滤条件
type LogListFilter struct {
	JobName string `bson:"jobName"`
}

//排序条件
type LogListSort struct {
	StartTimeOrder int64 `bson:"startTime"`
}

//单例
var (
	G_LogManage *LogManage
)

//根据条件获取日志列表
func (logMgr *LogManage) GetLogList(jobName string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var (
		cursor  mongo.Cursor
		filter  *LogListFilter
		sort    *LogListSort
		logInfo *common.JobLog
	)

	//实例化   不会返回nil
	logArr = make([]*common.JobLog, 0)

	//查询条件  任务名
	filter = &LogListFilter{JobName: jobName}
	//按开始时间倒序
	sort = &LogListSort{StartTimeOrder: -1}

	if cursor, err = logMgr.LogCollection.Find(context.TODO(), filter, findopt.Sort(sort), findopt.Skip(int64(skip)), findopt.Limit(int64(limit))); err != nil {
		return
	}
	//需要关闭游标
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		logInfo = &common.JobLog{}

		//mongo是bson，反序列化失败则跳过
		if err = cursor.Decode(logInfo); err != nil {
			log.Println(err)
			continue
		}

		logArr = append(logArr, logInfo)
	}

	return
}

//初始化日志管理
func InitLogManage() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(
		context.TODO(),
		G_Config.MongodbUri,
		clientopt.ConnectTimeout(time.Duration(G_Config.MongodbTimeOut)*time.Millisecond)); err != nil {
		return
	}

	G_LogManage = &LogManage{
		Client:        client,
		LogCollection: client.Database("cron").Collection("log"),
	}

	return
}
