package main

import (
	"airlch/crontab/master"
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
	flag.StringVar(&confFile, "config", "./crontab/master/main/master.json", "指定master.json")
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
	if err = master.InitConfig(confFile); err != nil {
		goto ERR
	}

	//初始化日志操作
	if err = master.InitLogManage(); err != nil {
		goto ERR
	}

	//任务管理器
	if err = master.InitJobManage(); err != nil {
		goto ERR
	}

	//初始化服务发现
	if err = master.InitWorkerManage(); err != nil {
		goto ERR
	}

	//启动api http服务
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(1 * time.Second)
	}

	//正常退出
	return
ERR:
	log.Println(err)
}
