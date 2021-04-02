package main

import (
	"fmt"
	"net"
)

func main() {
	NetTcp()
}

func NetTcp() {
	fmt.Println("Tcp net start")
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8008")
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Println("listen err=", err)
		return
	}
	defer server.Close()

	for {
		fmt.Println("wating...")
		conn, err := server.AcceptTCP()
		if err != nil {
			fmt.Println("AcceptTCP() err")
			continue
		}
		conn.SetKeepAlive(true)
		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		fmt.Println("client ip:", ip)

		go Process(conn)
	}
}
