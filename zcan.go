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

func index00(data []byte, start int) int {
	fmt.Println(data)
	pos := start
	for b := start; b < len(data); b++ {
		if data[b] == 0x00 {
			return b
		}
	}
	return pos
}

var dev *zcan.ZehnderDevice

func logModelData(rmi *zcan.ZehnderRMI) {
	serial, err := rmi.GetData(zcan.CN_STRING)
	if err != nil {
		fmt.Println(err)
		return
	}
	vers, err := rmi.GetData(zcan.CN_VERSION)
	if err != nil {
		fmt.Println(err)
		return
	}
	model, err := rmi.GetData(zcan.CN_STRING)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("Processing data for %s [%s] Version %d.%d", model, serial, vers.([]int)[0], vers.([]int)[1])
	dev.Name = fmt.Sprintf("%s [%s]", model, serial)
}

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

	dev.Start()

	if dumpFilename != "" {
		dev.SetDefaultRMICallback(storeRMI)
		fmt.Printf("Processing dumpfile: %s\n", dumpFilename)
		dev.ProcessDumpFile(dumpFilename)
		dev.Stop()
		dumpStoredRMI()
	} else {
		fmt.Printf("\n\nProcessing CAN packets. CTRL+C to quit...\n\n")

		dest := zcan.NewZehnderDestination(1, 1, 1)
		dest.GetMultiple(dev, []byte{4, 6, 8}, zcan.ZehnderRMITypeActualValue, logModelData)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigs
			dev.Stop()
		}()
	}

	dev.Wait()

	dev.DumpPDO()
}
