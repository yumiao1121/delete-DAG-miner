package main

import (
	"TCP_ceshi/model"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func Process(conn *net.TCPConn) {

	connbuff := bufio.NewReaderSize(conn, 1024)
	defer conn.Close()
	for {
		data, isPrefix, err := connbuff.ReadLine()
		fmt.Println("success")
		if isPrefix {
			fmt.Println("success1")
			log.Printf("Socket flood detected from")
			return
		} else if err == io.EOF {
			fmt.Println("success2")
			log.Printf("Client disconnected")
			break
		} else if err != nil {
			fmt.Println("success3")
			log.Printf("Error reading from socket: %v", err)
			return
		}

		if len(data) > 1 {
			var req model.JSONRpcReq

			err = json.Unmarshal(data, &req)
			fmt.Println("req:", req)
			if err != nil {
				log.Printf("Malformed stratum request from %v", err)
				return
			}
			_, err = handleTCPMessage(conn, &req)
			if err != nil {
				return
			}
		}
		fmt.Println("conn")
	}
}

func handleTCPMessage(conn *net.TCPConn, req *model.JSONRpcReq) (bool, error) {
	switch req.Method {
	case "eth_submitLogin":
		var params []string
		err := json.Unmarshal(req.Params, &params)
		fmt.Println(params)
		if err != nil {
			log.Println("Malformed stratum request params from")
			return false, err
		}
		reply, err := handleLoginRPC(params)
		fmt.Println(reply, err)
		if err != nil {
			return false, err
		}
		return false, sendTCPResult(conn, req.Id, reply)
	}
	return true, nil
}

func handleLoginRPC(params []string) (bool, error) {
	login := strings.ToLower(params[0])
	fmt.Println(login, "login")
	return true, nil
}

func sendTCPResult(conn *net.TCPConn, id json.RawMessage, result interface{}) error {
	enc := json.NewEncoder(conn)
	message := model.JSONRpcResp{Id: id, Version: "2.0", Error: nil, Result: result}
	data, _ := json.Marshal(message)

	fmt.Println(message, len(data))
	return enc.Encode(&message)
}
