package worker

import (
	"context"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"time"
)

//日志保存
type LogSink struct {
	Client        *mongo.Client
	LogCollection *mongo.Collection
}

//单例
var (
	G_LogSink *LogSink
)

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
	}

	return
}
