package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"paidpiper.com/payment-gateway/common"

	"paidpiper.com/payment-gateway/serviceNode"
)

var tracerShutdownFunc func()

func initGlobalTracer(url string, serviceName string) (*sdktrace.Provider, func()) {

	// Create and install Jaeger export pipeline
	provider, flush, err := jaeger.NewExportPipeline(
		// http://192.168.162.128:14268/api/traces
		jaeger.WithCollectorEndpoint(url),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: serviceName,
			Tags: []core.KeyValue{
				key.String("exporter", "jaeger"),
			},
		}),
		/// jaeger.RegisterAsGlobal() creates a lot of noise because of net/http traces, use it only if you really have to

		//jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),

		// NeverSample disables sampling
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.NeverSample()}),
	)

	if err != nil {
		log.Print("Could not connect to jaeger: " + err.Error())
	}

	_ = flush

	//return provider, flush
	return provider, nil
}

// _AttachParentProcess is a prent pid to attach to.
const _AttachParentProcess = ^uint32(0) // (DWORD)-1

var (
	kernel32DLL       = syscall.NewLazyDLL("kernel32.dll")
	attachConsoleProc = kernel32DLL.NewProc("AttachConsole")
	allocConsoleProc  = kernel32DLL.NewProc("AllocConsole")
	freeConsoleProc   = kernel32DLL.NewProc("FreeConsole")
)

// AttachConsole is a win api wrapper for AttachConsole function.
func AttachConsole(pid uint32) bool {
	ret, _, _ := syscall.Syscall(attachConsoleProc.Addr(), 1, uintptr(pid), 0, 0)
	return ret != 0
}

// AllocConsole is a win api wrapper for AllocConsole function.
func AllocConsole() bool {
	ret, _, _ := syscall.Syscall(allocConsoleProc.Addr(), 0, 0, 0, 0)
	return ret != 0
}

// FreeConsole is a win api wrapper for FreeConsole function.
func FreeConsole() bool {
	ret, _, _ := syscall.Syscall(freeConsoleProc.Addr(), 0, 0, 0, 0)
	return ret != 0
}

func main() {
	c := flag.Bool("console", false, "pass this flag in order to work in console")
	flag.Parse()
	if *c {
		if attached := AttachConsole(_AttachParentProcess); !attached {
			if allocated := AllocConsole(); !allocated {
				panic("cannot allocate a console")
			}
			defer FreeConsole()
		}
		stdOut, stdOutErr := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
		stdErr, stdErrErr := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
		if stdOutErr != nil || stdErrErr != nil {
			panic("cannot redirect stsandard output and error streams")
		}
		_, err := os.Stdout.Stat()
		if err != nil {
			os.Stdout = os.NewFile(uintptr(stdOut), "/dev/stdout")
		}
		_, err = os.Stderr.Stat()
		if os.Stderr == nil {
			os.Stderr = os.NewFile(uintptr(stdErr), "/dev/stderr")
		}
		oldArgs := os.Args
		os.Args = append([]string{}, oldArgs[0])
		os.Args = append(os.Args, oldArgs[2:]...)
	}

	config, err := common.ParseConfiguration("config.json")

	if err != nil {
		log.Print("Error reading configuration file (config.json), trying cmdline params: " + err.Error())

		if len(os.Args) < 3 {
			log.Panic("Reading configuration file failed, and no command line parameters supplied.")
		}

		config.StellarSeed = os.Args[1]
		config.AutoFlushPeriod = 15 * time.Minute
		config.MaxConcurrency = 10
		config.JaegerUrl = "http://192.168.162.128:14268/api/traces"
		config.JaegerServiceName = "PaymentGatewayTest"
		config.Port, err = strconv.Atoi(os.Args[2])

		if err != nil {
			log.Fatal("Port supplied, but couldn't be parsed")
		}
	}

	traceProvider, tracerShutdownFunc := initGlobalTracer(config.JaegerUrl, config.JaegerServiceName)
	common.InitializeTracer(traceProvider)

	//s := "SC2SCPAPTSPITDLJYR5WQRH23XK267D2KM5SFMUKBCVKSLI3TVFNEQHQ"
	//port := 28080

	runtime.GOMAXPROCS(config.MaxConcurrency)
	runtime.NumGoroutine()

	// Set up signal channel
	stop := make(chan os.Signal, 1)

	server, _, err := serviceNode.StartServiceNode(config.StellarSeed, config.Port, "http://localhost:5817", true, config.AutoFlushPeriod, config.TransactionValidityPeriodSec)

	if err != nil {
		log.Panicf("Error starting serviceNode: %v", err.Error())
	}

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Print("Error shutting down server: %v", err.Error())
	}

	tracerShutdownFunc()
}
