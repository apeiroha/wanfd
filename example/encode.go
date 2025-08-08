package main

import (
	"os"

	"github.com/WJQSERVER/wanf"
)

type Config struct {
	Server   Server   `wanf:"server"`
	Database Database `wanf:"database"`
	Name     string   `wanf:"name"`
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
	}

	// 创建打开文件
	f, err := os.Create("example.wanf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 生成文件example
	err = wanf.NewEncoder(f, wanf.WithStyle(wanf.StyleSingleLine)).Encode(cfg)
	if err != nil {
		panic(err)
	}

}
