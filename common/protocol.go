package common

import "encoding/json"

//任务模型
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
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
