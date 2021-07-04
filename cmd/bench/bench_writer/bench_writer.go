package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lim-team/LiMaoIM/pkg/client"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"go.uber.org/zap"
)

var (
	runfor     = flag.Duration("runfor", 10*time.Second, "duration of time to run")
	size       = flag.Int("size", 200, "size of messages")
	batchSize  = flag.Int("batch-size", 200, "batch size of messages")
	tcpAddress = flag.String("limao-tcp-address", "tcp://127.0.0.1:7677", "tpc://<addr>:<port> to connect to nsqd")
)

var totalMsgCount int64

func main() {
	flag.Parse()
	var wg sync.WaitGroup
	msg := make([]byte, *size)
	batch := make([][]byte, *batchSize)
	for i := range batch {
		batch[i] = msg
	}
	goChan := make(chan int)
	readyChan := make(chan int)
	for j := 0; j < runtime.GOMAXPROCS(0); j++ {
		wg.Add(1)
		go func() {
			sendWorker(fmt.Sprintf("test-%d", j), *runfor, *tcpAddress, batch, readyChan, goChan)
			wg.Done()
		}()
		<-readyChan
	}
	start := time.Now()
	close(goChan)
	wg.Wait()
	end := time.Now()
	duration := end.Sub(start)
	tmc := atomic.LoadInt64(&totalMsgCount)
	log.Printf("duration: %s - %.03fmb/s - %.03fops/s - %.03fus/op",
		duration,
		float64(tmc*int64(*size))/duration.Seconds()/1024/1024,
		float64(tmc)/duration.Seconds(),
		float64(duration/time.Microsecond)/float64(tmc))
}

func sendWorker(uid string, td time.Duration, tcpAddr string, batch [][]byte, readyChan chan int, goChan chan int) {
	c := client.New(tcpAddr, client.WithUID(uid))
	err := c.Connect()
	if err != nil {
		panic(err)
	}
	var msgCount int64
	endTime := time.Now().Add(td)
	readyChan <- 1
	<-goChan

	var wg sync.WaitGroup

	c.SetOnSendack(func(sendack *lmproto.SendackPacket) {
		wg.Done()
	})
	for {
		for _, b := range batch {
			wg.Add(1)
			err := c.SendMessage(client.NewChannel("toTest2", 1), b, client.SendOptionWithFlush(false))
			if err != nil {
				fmt.Println("send message fail", zap.Error(err))
			}
		}
		err = c.Flush()
		if err != nil {
			fmt.Println("flush message fail", zap.Error(err))
		}
		msgCount += int64(len(batch))
		if time.Now().After(endTime) {
			break
		}
	}
	wg.Wait()
	atomic.AddInt64(&totalMsgCount, msgCount)
}
