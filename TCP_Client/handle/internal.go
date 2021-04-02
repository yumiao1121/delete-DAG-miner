package handle

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type login struct {
	Id     int         `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type reply struct {
	Id      interface{} `json:"id"`
	Version string      `json:"version"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

func NewConnTcp(conn net.Conn, id int, address string, ip string) ConnInfo {
	return ConnInfo{
		Conn:    conn,
		Id:      id,
		Ip:      ip,
		Address: address,
	}
}

//如果id是0说明是下发任务的消息
func (w *Worker) NewTcpLogin() bool {
	RequstJson := map[string]interface{}{"id": w.Conntcp.Id, "method": "eth_submitLogin", "Version": "2.0", "params": []string{w.Conntcp.Address}}
	data, _ := json.Marshal(RequstJson)
	fmt.Println("登陆请求：", string(data))

	n, err := w.Conntcp.Conn.Write(data)
	w.Conntcp.Conn.Write([]byte("\n"))

	if err != nil {
		fmt.Println("write dial err=", err)
		return false
	}

	buf := make([]byte, 8096)

	n, err = w.Conntcp.Conn.Read(buf)
	if err != nil {
		fmt.Println("Login responde Read error")
		return false
	}

	var result reply
	err = json.Unmarshal(buf[:n], &result)
	if err != nil {
		fmt.Println("Login responde json.Unmarsha err=", err)
		return false
	}
	//elapsed := time.Since(t1)

	//log.Printf("Login responde Id:%v\tVersion:%v\tResult:%v\tError:%v\n", result.Id, result.Version, result.Result, result.Error)
	log.Println(result)
	if result.Error != nil {
		fmt.Println("id：", w.Conntcp.Id, "login err:", result.Error)
		return false
	}
	if slice, ok := result.Result.(bool); ok {
		if slice == true {
			fmt.Println("id：", w.Conntcp.Id, "login success")
			return true
		}
	}
	return false
}

func (w *Worker) NewTcpGetWork() bool {
	RequstJson := map[string]interface{}{"id": 1, "method": "eth_getWork", "Version": "2.0"}
	data, _ := json.Marshal(RequstJson)
	_, err := w.Conntcp.Conn.Write(data)
	w.Conntcp.Conn.Write([]byte("\n"))
	if err != nil {
		fmt.Println("Get work write err=", err)
		return false
	}
	return true
}

func (w *Worker) NewTcpSubmitWork(mixdigest []byte, nonce uint64, hash string) {
	//newHash := []byte(w.Work.Hash)
	//fmt.Println("send:", fmt.Sprintf("0x%016x", nonce), w.Work.Hash, "0x"+hex.EncodeToString(mixdigest))
	//fmt.Println(hash)
	//nonce1, _ := strconv.ParseUint(strings.Replace(fmt.Sprintf("0x%016x", nonce), "0x", "", -1), 16, 64)
	//fmt.Println(nonce, nonce1)
	RequstJson := map[string]interface{}{"id": w.Conntcp.Id, "method": "eth_submitWork",
		"Version": "2.0", "params": []string{fmt.Sprintf("0x%016x", nonce), hash, "0x" + hex.EncodeToString(mixdigest)},
	}

	data, _ := json.Marshal(RequstJson)
	_, err := w.Conntcp.Conn.Write(data)
	w.Conntcp.Conn.Write([]byte("\n"))
	if err != nil {
		fmt.Println("Submit Work write dial err=", err)
		return
	}

}
