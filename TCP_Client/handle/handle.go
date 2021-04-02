package handle

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"strconv"
)

type Worker struct { //每个节点对应一个work
	//sync.Mutex
	Conntcp ConnInfo
	Work    Work
}

type Work struct { //接受矿池下发的工作保存
	Hash   string
	Target *big.Int
	Seed   uint64
}

type ConnInfo struct { //节点信息
	Conn    net.Conn
	Address string
	Id      int
	Ip      string
}

func NewWorker(id int, exit chan bool) {
	conn, err := net.Dial("tcp", "0.0.0.0:8008") //连接矿池
	if err != nil {
		log.Println("connect Server dial err=", err)
		exit <- true
		return
	}
	defer conn.Close()

	conntcp := NewConnTcp(conn, id, "0x6c9019e157adb466f353498ed1bacf8a95f3544c", "0.0.0.0")
	worker := Worker{
		Conntcp: conntcp,
	}
	//计算share target
	two256 := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	target := new(big.Int).Div(two256, big.NewInt(1000))

	work := Work{
		Target: target,
	}
	worker.Work = work
	if !worker.NewTcpLogin() {
		exit <- true
		return
	}

	Start(&worker)
}

func Start(worker *Worker) {
	nodeover := make(chan bool)
	err := worker.NewTcpGetWork()
	if !err {
		log.Println("newTcpGetwork err")
	}
	abort := make(chan bool, 10)

	go worker.MessageDistrbution(abort, nodeover)
	go worker.Miner(abort)

	<-nodeover
}

func (w *Worker) MessageDistrbution(abort chan bool, nodeover chan bool) { //消息分发
	for {
		buf := make([]byte, 8096)
		n, err := w.Conntcp.Conn.Read(buf)
		if err != nil {
			fmt.Println("Getwork Read error:", err)
			nodeover <- true
			break
		}

		var result reply
		err = json.Unmarshal(buf[:n], &result)
		if err != nil {
			log.Println("json.Unmarsha err=", err)
			continue
		}

		if slice, ok := result.Result.([]interface{}); ok { //类型断言，将接收到的消息断言为inferface{}类型
			log.Println(result)
			if len(slice) == 3 {
				if slice[0] != w.Work.Hash {
					res, _ := slice[0].(string) //区块hash
					w.Work.Hash = res

					num, _ := slice[1].(string) //区块高度
					//fmt.Println("res:", res, "height:", num)
					Uintnum, _ := strconv.Atoi(num)
					w.Work.Seed = uint64(Uintnum)
					abort <- true //chan 更新work后在挖
				}
			}
		} else {
			log.Println("return result:", result)
		}
	}

}
