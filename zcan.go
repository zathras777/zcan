package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zathras777/zcan/pkg/zcan"
)

var dev *zcan.ZehnderDevice

var rmiMsgs []*zcan.ZehnderRMI

func storeRMI(rmi *zcan.ZehnderRMI) {
	rmiMsgs = append(rmiMsgs, rmi)
}

func dumpStoredRMI() {
	fmt.Println("\nRMI Messages")
	for _, rmi := range rmiMsgs {
		fmt.Printf("%08X : Source %d Dest %d Counter %d  Sequence %d\n", rmi.MakeCANId(), rmi.SourceId,
			rmi.DestId, rmi.Counter, rmi.Sequence)
		fmt.Printf("         : IsMulti %t  IsRequest %t  IsError %t\n", rmi.IsMulti, rmi.IsRequest, rmi.IsError)
		fmt.Printf("         : %d bytes %v\n", rmi.DataLength, rmi.Data[:rmi.DataLength])
	}
}

func requestPDO(dev *zcan.ZehnderDevice) {
	type pdoToRequest struct {
		productId byte
		pdoId     uint16
		frequency byte
	}
	var requests = []pdoToRequest{
		{1, 49, 0xff},
		{1, 65, 0xff},
		{1, 81, 1},
		{1, 117, 5},
		{1, 118, 5},
		{1, 119, 5},
		{1, 120, 5},
		{1, 121, 5},
		{1, 122, 5},
		{1, 192, 0xff},
		{1, 209, 0xff},
		{1, 227, 0x10},
		{1, 274, 2},
		{1, 275, 2},
		{1, 276, 2},
		{1, 278, 2},
	}
	for _, req := range requests {
		dev.RequestPDO(req.productId, req.pdoId, req.frequency)
	}
	dev.RequestPDOBySlug(1, "exhaust_humidity", 2)

}

func main() {
	var (
		nodeId       int
		dumpFilename string
		intName      string
		captureAll   bool
		captureFn    string
		host         string
		port         int
	)

	flag.IntVar(&nodeId, "nodeid", 55, "Node ID to use for client")
	flag.StringVar(&dumpFilename, "dumpfile", "", "Dump file to process")
	flag.StringVar(&intName, "interface", "", "CAN Network Interface name")
	flag.BoolVar(&captureAll, "capture", false, "Capture all CAN packets for debugging")
	flag.StringVar(&captureFn, "capture-filename", "output", "Capture filename [default: output]")
	flag.IntVar(&port, "port", 7004, "Port for HTTP server")
	flag.StringVar(&host, "address", "127.0.0.1", "Address for HTTP server")
	flag.Parse()

	if dumpFilename == "" && intName == "" {
		fmt.Println("Nothing to do. Specify either a dump filename or interface name.")
		return
	}

	dev = zcan.NewZehnderDevice(byte(nodeId & 0xff))
	if intName != "" {
		if err := dev.Connect(intName); err != nil {
			fmt.Println(err)
			return
		}
		dev.StartHttpServer(host, port)
	}
	if captureAll {
		if dumpFilename != "" {
			fmt.Println("Cannot capture and parse a dump file at the same time. Ignoring capture request.")
		} else {
			dev.CaptureAll(captureFn)
		}
	}

	if dumpFilename == "" {
		f, err := os.OpenFile("zcan.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

	dev.Start()

	if dumpFilename != "" {
		dev.SetDefaultRMICallback(storeRMI)
		fmt.Printf("Processing dumpfile: %s\n", dumpFilename)
		dev.ProcessDumpFile(dumpFilename)
		dev.Stop()
		dumpStoredRMI()
	} else {
		fmt.Printf("\n\nProcessing CAN packets. CTRL+C to quit...\n\n")
		requestPDO(dev)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigs
			dev.Stop()
		}()
	}
	fmt.Println("Waiting for everything to complete...")
	dev.Wait()

	dev.DumpPDO()
}
