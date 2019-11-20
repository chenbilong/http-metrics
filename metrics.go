// Gor is simple http traffic replication tool written in Go. Its main goal to replay traffic from production servers to staging and dev environments.
// Now you can test your code on real user sessions in an automated and repeatable fashion.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	_ "runtime/debug"
	"syscall"
	"time"
)

var closeCh chan int

func main() {
	closeCh = make(chan int)
	// // Don't exit on panic
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Printf("PANIC: pkg: %v %s \n", r, debug.Stack())
	// 	}
	// }()

	// If not set via env cariable
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	}

	flag.Parse()
	InitPlugins()

	fmt.Println("Version:", VERSION)

	if len(Plugins.Inputs) == 0 || len(Plugins.Outputs) == 0 {
		log.Fatal("Required at least 1 input and 1 output")
	}

	if Settings.pprof != "" {
		go func() {
			log.Println(http.ListenAndServe(Settings.pprof, nil))
		}()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		finalize()
		os.Exit(1)
	}()

	if Settings.exitAfter > 0 {
		log.Println("Running gor for a duration of", Settings.exitAfter)

		time.AfterFunc(Settings.exitAfter, func() {
			log.Println("Stopping gor after", Settings.exitAfter)
			close(closeCh)
		})
	}

	Start(closeCh)
}

func finalize() {
	for _, p := range Plugins.All {
		if cp, ok := p.(io.Closer); ok {
			cp.Close()
		}
	}
}
