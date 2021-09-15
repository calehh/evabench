package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var WsCntFlag = cli.IntFlag{
	Name:   "num",
	Usage:  "websocket connection number",
	Hidden: false,
	Value:  0,
}

var UrlFlag = cli.StringFlag{
	Name:   "url",
	Usage:  "full node ws sever url",
	Hidden: false,
	Value:  "",
}

var ErrCount = int32(0)

var app *cli.App

func init() {
	app = cli.NewApp()
	app.Version = "v0.0.4"
	app.Flags = []cli.Flag{
		UrlFlag,
		WsCntFlag,
	}

}

func main() {
	app.Action = Start
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Start(ctx *cli.Context) {
	defer func() {
		fmt.Println("err count:", ErrCount)
	}()
	cnt := ctx.Int("num")
	url := ctx.String("url")
	for i := 0; i < cnt; i++ {
		index := i
		go func() {
			DialWS(index, url)
		}()
	}
	waitToExit()
}

func DialWS(index int, url string) {
	fmt.Println("dial", url)
	headerCh := make(chan *types.Header)
	client, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("Websocket dial Evanesco node err", "index:", index, "err:", err)
		atomic.AddInt32(&ErrCount, int32(1))
		return
	}
	sub, err := client.EthSubscribe(context.Background(), headerCh, "newHeads")
	if err != nil {
		fmt.Println("Subscribe block err", "err:", err)
		atomic.AddInt32(&ErrCount, int32(1))
		return
	}

	go func() {
		timer := time.NewTimer(time.Second * 10)
		for {
			select {
			case e := <-sub.Err():
				fmt.Println("sub channel err", "index:", index, "error:", e)
				atomic.AddInt32(&ErrCount, int32(1))
				client.Close()
				return
			case <-headerCh:
				timer.Reset(time.Second * 10)
			case <-timer.C:
				fmt.Println("new header timeout", "index:", index)
				atomic.AddInt32(&ErrCount, int32(1))
				client.Close()
				return
			}
		}
	}()
	return
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			fmt.Printf("received exit signal:%v\n", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
