package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zathras777/zcan/pkg/zcan"
)

func showSerialNumber(data []byte) {
	fmt.Printf("Test #1 -> Serial Number: %s\n", string(data))
}

func showModel(data []byte) {
	fmt.Printf("Test #2 -> Model Description: %s\n", string(data))
}

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

func logModelData(data []byte) {
	pos := index00(data, 0)
	serial := string(data[:pos])
	version := binary.LittleEndian.Uint32(data[pos+1 : pos+5])
	maj, min := zcan.ZehnderVersionDecode(version)
	model := string(data[pos+5 : index00(data, pos+5)])
	log.Printf("Processing data for %s [%s] Version %d.%d", model, serial, maj, min)
	dev.Name = fmt.Sprintf("%s [%s]", model, serial)
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
		fmt.Printf("Processing dumpfile: %s\n", dumpFilename)
		dev.ProcessDumpFile(dumpFilename)
		dev.Stop()
	} else {
		fmt.Printf("\n\nProcessing CAN packets. CTRL+C to quit...\n\n")

		dest := zcan.NewZehnderDestination(1, 1, 1)
		dest.GetOne(dev, 4, zcan.ZehnderRMITypeActualValue, showSerialNumber)
		dest.GetOne(dev, 8, zcan.ZehnderRMITypeActualValue, showModel)
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
