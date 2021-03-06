package main

import (
	"airlch/crontab/worker"
	"flag"
	"log"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

func initArgs() {
	//master -config ./master.json
	//master -h
	//flag.StringVar(&confFile,"config","./master.json","指定master.json")
	//flag.StringVar(&confFile, "config", "./crontab/worker/main/worker.json", "指定worker.json")
	flag.StringVar(&confFile, "config", "E:/own/go/src/airlch/crontab/worker/main/worker.json", "指定worker.json")
	//flag.StringVar(&confFile, "config", "E:/main/worker.json", "指定worker.json")
	flag.Parse()
}

func initEnv() {
	//根据cpu数量开启线程
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)

	//初始化配置文件
	initArgs()

	//初始化线程
	initEnv()

	//初始化配置
	if err = worker.InitConfig(confFile); err != nil {
		goto ERR
	}

	//初始化执行器
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}

	//初始化日志保存
	if err = worker.InitLogSink(); err != nil {
		goto ERR
	}

	//初始化调度器
	if err = worker.InitSchedule(); err != nil {
		goto ERR
	}

	//初始化服务注册
	if err = worker.InitRegist(); err != nil {
		goto ERR
	}

	//任务管理器
	if err = worker.InitJobManage(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(time.Second)
	}

	//正常退出
	return
ERR:
	log.Println(err)
}
