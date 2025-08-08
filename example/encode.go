package main

import (
	"os"

	"github.com/WJQSERVER/wanf"
)

type Config struct {
	Server   Server              `wanf:"server"`
	Database Database            `wanf:"database"`
	Name     string              `wanf:"name"`
	Info     []string            `wanf:"info"`
	DashMap  map[string]string   `wanf:"dashMap"`
	SList    map[string]struct{} `wanf:"sList"`
	DList    map[string]Detail   `wanf:"dList"`
}

type Server struct {
	Host string `wanf:"host"`
	Port int    `wanf:"port"`
}

type Database struct {
	User     string `wanf:"user"`
	Password string `wanf:"password"`
	Port     int    `wanf:"port"`
}

type Detail struct {
	ID   int    `wanf:"id"`
	Name string `wanf:"name"`
}

func main() {
	cfg := Config{
		Server: Server{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Database: Database{
			User:     "root",
			Password: "password",
			Port:     3306,
		},
		Name: "My App",
		Info: []string{"Info 1", "Info 2"},
		DashMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		SList: map[string]struct{}{
			"item1": {},
			"item2": {},
		},
		DList: map[string]Detail{
			"detail1": {ID: 1, Name: "Detail One"},
			"detail2": {ID: 2, Name: "Detail Two"},
		},
	}

	// 创建打开文件
	f, err := os.Create("example.wanf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 生成文件example
	err = wanf.NewEncoder(f, wanf.WithStyle(wanf.StyleDefault)).Encode(cfg)
	if err != nil {
		panic(err)
	}

}
