package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type reply struct {
	Id interface{} `json:"id"`
	//Id      interface{} `json:"id"`
	Version string      `json:"version"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

func main() {
	exit := make(chan bool, 0)
	times := 1000
	for i := 0; i < times; i++ {
		fmt.Println("connect:", i)
		go TCP_connect(i)
	}
	<-exit
}

func TCP_connect(i int) {
	conn, err := net.Dial("tcp", "0.0.0.0:8008") //连接矿池
	if err != nil {
		fmt.Println("client dial err=", err)
		return
	}
	defer conn.Close()
	RequstJson := map[string]interface{}{"id": i, "method": "eth_submitLogin", "Version": "2.0", "params": []string{"0xb85150eb365e7df0941f0cf08235f987ba91506a"}}
	data, _ := json.Marshal(RequstJson)
	fmt.Println("登陆请求：", string(data))

	n, err := conn.Write(data)
	conn.Write([]byte("\n"))
	if err != nil {
		fmt.Println("write dial err=", err)
	}
	buf := make([]byte, 8096)

	n, err = conn.Read(buf)
	if err != nil {
		fmt.Println("Login responde Read error")
	}

	var result reply
	err = json.Unmarshal(buf[:n], &result)
	if err != nil {
		fmt.Println("Login responde json.Unmarsha err=", err)
	}
	//elapsed := time.Since(t1)

	//log.Printf("Login responde Id:%v\tVersion:%v\tResult:%v\tError:%v\n", result.Id, result.Version, result.Result, result.Error)
	log.Println(result)
	if result.Error != nil {
	}
	if slice, ok := result.Result.(bool); ok {
		if slice == true {
			fmt.Println("id：", "login success")
		}
	}

	connbuf := bufio.NewReaderSize(conn, 8096)
	for {
		_, _, err := connbuf.ReadLine()
		if err != nil {
			fmt.Println("Getwork Read error:", err)
		}
	}

}
