package master

import (
	"airlch/crontab/common"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

//任务的http接口
type apiServer struct {
	httpServer *http.Server
}

//保存任务接口
// job:{"name":"job1","command":"echo hello","cronExpr":"* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	//任务保存到ETCD中
	var (
		err       error
		postjob   string
		jobObj    common.Job
		oldJob    *common.Job
		respBytes []byte
	)

	//1.解析post表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//2.获取表单中的job值
	postjob = req.PostForm.Get("job")

	//3.反序列化
	if err = json.Unmarshal([]byte(postjob), &jobObj); err != nil {
		goto ERR
	}

	// 4, 保存到etcd
	if oldJob, err = G_JobManage.SaveJob(&jobObj); err != nil {
		goto ERR
	}

	// 5, 返回正常应答 ({"errno": 0, "msg": "", "data": {....}})
	if respBytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(respBytes)
	}
	return

ERR:
	log.Println("handleJobSave:", err.Error())
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respBytes)
	}
}

//删除任务接口
//post /job/delete jobName=job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err       error
		jobName   string
		oldJob    *common.Job
		respBytes []byte
	)

	//1.解析post表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//2.获取表单中的jobName
	jobName = req.PostForm.Get("name")

	//3.删除操作
	if oldJob, err = G_JobManage.DeleteJob(jobName); err != nil {
		goto ERR
	}

	//4.返回应答信息
	if respBytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(respBytes)
	}

	return

ERR:
	log.Println("handleJobDelete:", err.Error())
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respBytes)
	}
}

//获取任务列表接口
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		err       error
		jobList   []*common.Job
		respBytes []byte
	)

	//3.获取列表
	if jobList, err = G_JobManage.GetJobList(); err != nil {
		goto ERR
	}

	//4.返回应答信息
	if respBytes, err = common.BuildResponse(0, "success", jobList); err == nil {
		resp.Write(respBytes)
	}

	return

ERR:
	log.Println("handleJobList:", err.Error())
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respBytes)
	}
}

//强杀任务操作
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err       error
		jobName   string
		respBytes []byte
	)

	//1.解析post表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	//2.获取表单中的jobName
	jobName = req.PostForm.Get("name")

	if len(jobName) == 0 {
		err = errors.New("请输入name")
		goto ERR
	}

	//3.强杀操作
	if err = G_JobManage.KillJob(jobName); err != nil {
		goto ERR
	}

	//4.返回应答信息
	if respBytes, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(respBytes)
	}

	return

ERR:
	log.Println("handleJobKill:", err.Error())
	if respBytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respBytes)
	}
}

var (
	//单例对象
	G_ApiServer *apiServer
)

//初始化服务
func InitApiServer() (err error) {
	var (
		mux          *http.ServeMux
		listener     net.Listener
		httpServer   *http.Server
		staticDir    http.Dir
		staticHandle http.Handler
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)

	//配置静态资源路由，前端路由

	//根目录路径
	staticDir = http.Dir(G_Config.WebRoot)
	//创建静态文件handle
	staticHandle = http.FileServer(staticDir)

	//http.StripPrefix   截掉该部分   例子 访问：/index.html   截掉/，得到的是index.html
	mux.Handle("/", http.StripPrefix("/", staticHandle))

	//启动tcp监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_Config.ApiPort)); err != nil {
		return err
	}

	//创建一个http服务
	httpServer = &http.Server{
		//读，写 5秒超时
		ReadTimeout:  time.Duration(G_Config.ApiReadTimeOut) * time.Millisecond,
		WriteTimeout: time.Duration(G_Config.ApiWriteTimeOut) * time.Millisecond,
		//配置路由，其实也是一个handle
		Handler: mux,
	}

	//启动了服务端
	go httpServer.Serve(listener)

	//赋值单例
	G_ApiServer = &apiServer{
		httpServer: httpServer,
	}

	return
}
