package handle

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"strconv"
)

type Worker struct {
	//sync.Mutex
	Conntcp ConnInfo
	Work    Work
}
type Work struct {
	Hash   string
	Target *big.Int //
	Seed   uint64
}

func NewWorker(id int, exit chan bool) {
	conn, err := net.Dial("tcp", "0.0.0.0:8008") //连接矿池
	if err != nil {
		fmt.Println("client dial err=", err)
		return
	}
	defer conn.Close()
	conntcp := NewConnTcp(conn, id, "0xb85150eb365e7df0941f0cf08235f987ba91506a", "0.0.0.0")
	worker := Worker{
		Conntcp: conntcp,
	}

	two256 := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	//	diff, err1 := new(big.Int).SetString("4096", 0)
	// if !err1 {
	// 	fmt.Println("diff compute err")
	// 	return
	// }
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

	go worker.MessageDistrbution(abort)
	go worker.Miner(abort)

	<-nodeover
}

func (w *Worker) MessageDistrbution(abort chan bool) {
	for {
		buf := make([]byte, 8096)
		n, err := w.Conntcp.Conn.Read(buf)
		if err != nil {
			fmt.Println("Getwork Read error:", err)
			//fmt.Println("buf len", buf)
			//continue
			break
		}

		var result reply
		err = json.Unmarshal(buf[:n], &result)
		if err != nil {
			fmt.Println("json.Unmarsha err=", err)
		}

		if slice, ok := result.Result.([]interface{}); ok {
			fmt.Println(result)
			if len(slice) == 3 {
				if slice[0] != w.Work.Hash {
					// res, _ := slice[1].(string)
					// seed, err := strconv.Atoi(res) //0:hash 1:seed 2:diff
					// w.Work.Seed = uint64(seed)
					// if err != nil {
					// 	fmt.Println("w.Work.seed transformation err")
					// 	return
					// }
					res, _ := slice[0].(string)
					w.Work.Hash = res
					num, _ := slice[1].(string)
					fmt.Println("res:", res, "height:", num)
					Uintnum, _ := strconv.Atoi(num)
					w.Work.Seed = uint64(Uintnum)
					abort <- true
				}
			}
		} else {
			fmt.Println("result:", result)
		}
	}

}
