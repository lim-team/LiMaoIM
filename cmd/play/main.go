package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lim-team/LiMaoIM/pkg/client"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/tangtaoit/limnet/pkg/limutil/sync"
)

var (
	user   = flag.String("user", "", "user id")
	tokenV = flag.String("token", "testtoken", "user token")
	addrV  = flag.String("addr", "127.0.0.1:7677", "server address E.g：127.0.0.1:7677")
)

func main() {
	flag.Parse()

	ch := make(chan string, 16)
	uid := *user
	token := *tokenV
	addr := *addrV
	if uid == "" {
		fmt.Println("user不能为空！")
		return
	}
	if token == "" {
		token = "testtoken"
	}

	cl := client.New(fmt.Sprintf("tcp://%s", addr), client.WithAutoReconn(true), client.WithUID(uid), client.WithToken(token))

	err := cl.Connect()
	if err != nil {
		fmt.Println("connect fail,", err.Error())
		return
	}

	cl.SetOnRecv(func(recv *lmproto.RecvPacket) error {
		fmt.Println(recv.FromUID, ":", string(recv.Payload))
		fmt.Printf(">")
		return nil
	})

	waitWrap := sync.WaitGroupWrapper{}

	waitWrap.AddAndRun(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf(">")
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				break
			}
			ch <- s
		}
	})

	waitWrap.AddAndRun(func() {
		for {
			select {
			case msg := <-ch:
				cmd, p1, p2 := splitCMD(msg)
				if cmd == "send" {
					err = cl.SendMessage(client.NewChannel(p2, 1), []byte(p1))
					if err != nil {
						fmt.Println("发送消息失败！", err.Error())
						continue
					}
				}

			}
		}
	})

	waitWrap.Wait()

}

func splitCMD(v string) (string, string, string) {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "send") && strings.Index(v, " to ") != -1 { // send
		toPos := strings.Index(v, " to ")
		channel := strings.TrimSpace(v[toPos+len(" to "):])
		msg := strings.TrimSpace(v[len("send"):toPos])
		return "send", msg, channel
	}
	return "", "", ""
}
