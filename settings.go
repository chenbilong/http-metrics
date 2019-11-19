package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

var VERSION string

// MultiOption allows to specify multiple flags with same name and collects all values into array
type MultiOption []string

func (h *MultiOption) String() string {
	return fmt.Sprint(*h)
}

// Set gets called multiple times for each flag with same name
func (h *MultiOption) Set(value string) error {
	*h = append(*h, value)
	return nil
}

// AppSettings is the struct of main configuration
type AppSettings struct {
	verbose   bool
	debug     bool
	stats     bool
	exitAfter time.Duration

	pprof string

	splitOutput bool

	inputDummy   MultiOption
	outputDummy  MultiOption
	outputStdout bool
	outputNull   bool

	inputRAW                MultiOption
	inputRAWEngine          string
	inputRAWTrackResponse   bool
	inputRAWRealIPHeader    string
	inputRAWExpire          time.Duration
	inputRAWBpfFilter       string
	inputRAWTimestampType   string
	copyBufferSize          int
	inputRAWImmediateMode   bool
	inputRawBufferSize      int
	inputRAWOverrideSnapLen bool

	prettifyHTTP bool
}

// Settings holds Gor configuration
var Settings AppSettings

func usage() {
	fmt.Printf("Gor is a simple http traffic replication tool written in Go. Its main goal is to replay traffic from production servers to staging and dev environments.\nProject page: https://github.com/buger/gor\nAuthor: <Leonid Bugaev> leonsbox@gmail.com\nCurrent Version: %s\n\n", VERSION)
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	flag.Usage = usage

	flag.StringVar(&Settings.pprof, "http-pprof", "", "Enable profiling. Starts  http server on specified port, exposing special /debug/pprof endpoint. Example: `:8181`")
	flag.BoolVar(&Settings.verbose, "verbose", false, "Turn on more verbose output")
	flag.BoolVar(&Settings.debug, "debug", false, "Turn on debug output, shows all intercepted traffic. Works only when with `verbose` flag")
	flag.BoolVar(&Settings.stats, "stats", false, "Turn on queue stats output")
	flag.DurationVar(&Settings.exitAfter, "exit-after", 0, "exit after specified duration")

	flag.BoolVar(&Settings.splitOutput, "split-output", false, "By default each output gets same traffic. If set to `true` it splits traffic equally among all outputs.")

	flag.Var(&Settings.inputDummy, "input-dummy", "Used for testing outputs. Emits 'Get /' request every 1s")
	flag.Var(&Settings.outputDummy, "output-dummy", "DEPRECATED: use --output-stdout instead")

	flag.BoolVar(&Settings.outputStdout, "output-stdout", false, "Used for testing inputs. Just prints to console data coming from inputs.")

	flag.BoolVar(&Settings.outputNull, "output-null", false, "Used for testing inputs. Drops all requests.")

	flag.BoolVar(&Settings.prettifyHTTP, "prettify-http", false, "If enabled, will automatically decode requests and responses with: Content-Encodning: gzip and Transfer-Encoding: chunked. Useful for debugging, in conjuction with --output-stdout")

	flag.Var(&Settings.inputRAW, "input-raw", "Capture traffic from given port (use RAW sockets and require *sudo* access):\n\t# Capture traffic from 8080 port\n\tgor --input-raw :8080 --output-http staging.com")

	flag.BoolVar(&Settings.inputRAWTrackResponse, "input-raw-track-response", false, "If turned on Gor will track responses in addition to requests, and they will be available to middleware and file output.")

	flag.StringVar(&Settings.inputRAWEngine, "input-raw-engine", "libpcap", "Intercept traffic using `libpcap` (default), and `raw_socket`")

	flag.StringVar(&Settings.inputRAWRealIPHeader, "input-raw-realip-header", "", "If not blank, injects header with given name and real IP value to the request payload. Usually this header should be named: X-Real-IP")

	flag.DurationVar(&Settings.inputRAWExpire, "input-raw-expire", time.Second*2, "How much it should wait for the last TCP packet, till consider that TCP message complete.")

	flag.StringVar(&Settings.inputRAWBpfFilter, "input-raw-bpf-filter", "", "BPF filter to write custom expressions. Can be useful in case of non standard network interfaces like tunneling or SPAN port. Example: --input-raw-bpf-filter 'dst port 80'")

	flag.StringVar(&Settings.inputRAWTimestampType, "input-raw-timestamp-type", "", "Possible values: PCAP_TSTAMP_HOST, PCAP_TSTAMP_HOST_LOWPREC, PCAP_TSTAMP_HOST_HIPREC, PCAP_TSTAMP_ADAPTER, PCAP_TSTAMP_ADAPTER_UNSYNCED. This values not supported on all systems, GoReplay will tell you available values of you put wrong one.")
	flag.IntVar(&Settings.copyBufferSize, "copy-buffer-size", 5*1024*1024, "Set the buffer size for an individual request (default 5M)")
	flag.BoolVar(&Settings.inputRAWOverrideSnapLen, "input-raw-override-snaplen", false, "Override the capture snaplen to be 64k. Required for some Virtualized environments")
	flag.BoolVar(&Settings.inputRAWImmediateMode, "input-raw-immediate-mode", false, "Set pcap interface to immediate mode.")

	flag.IntVar(&Settings.inputRawBufferSize, "input-raw-buffer-size", 0, "Controls size of the OS buffer (in bytes) which holds packets until they dispatched. Default value depends by system: in Linux around 2MB. If you see big package drop, increase this value.")

	// flag.Var(&Settings.inputHTTP, "input-http", "Read requests from HTTP, should be explicitly sent from your application:\n\t# Listen for http on 9000\n\tgor --input-http :9000 --output-http staging.com")

}

var previousDebugTime int64
var debugMutex sync.Mutex

// Debug gets called only if --verbose flag specified
func Debug(args ...interface{}) {
	if Settings.verbose {
		debugMutex.Lock()
		now := time.Now()
		diff := float64(now.UnixNano()-previousDebugTime) / 1000000
		previousDebugTime = now.UnixNano()
		debugMutex.Unlock()

		fmt.Printf("[DEBUG][PID %d][%d][%fms] ", os.Getpid(), now.UnixNano(), diff)
		fmt.Println(args...)
	}
}
