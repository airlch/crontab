package worker

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	EtcdEndpoints    []string `json:"etcdEndpoints"`
	EtcdDialTimeout  int      `json:"etcdDialTimeout"`
	ExecuteWorkCount int      `json:"executeWorkCount"`
}

var (
	G_Config *Config
)

func InitConfig(fileName string) (err error) {
	var (
		content []byte
		conf    Config
	)

	//1.读取json配置文件
	if content, err = ioutil.ReadFile(fileName); err != nil {
		return
	}

	//2.反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	//3.赋值单例
	G_Config = &conf

	return
}
