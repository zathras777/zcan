package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zathras777/zcan/pkg/zcan"
)

func showSerialNumber(data []byte) {
	fmt.Printf("Test #1 -> Serial Number: %s\n", string(data))
}

func showModel(data []byte) {
	fmt.Printf("Test #2 -> Model DEscription: %s\n", string(data))
}

func index00(data []byte, start int) int {
	pos := start
	for b := start; b < len(data); b++ {
		if data[b] == 0x00 {
			return b
		}
	}
	return pos
}

func showMultiple(data []byte) {
	fmt.Println("Test #3 - Multiple")
	pos := index00(data, 0)
	serial := string(data[:pos])
	version := binary.LittleEndian.Uint32(data[pos+1 : pos+5])
	maj, min := zcan.ZehnderVersionDecode(version)
	model := string(data[pos+5:])
	fmt.Printf("\tSerial Number: %s\n\tVersion: %d -> [%d.%d]\n\tModel: %s\n\n", serial, version, maj, min, model)
}

func main() {
	var (
		nodeId       int
		dumpFilename string
		intName      string
		captureAll   bool
		captureFn    string
	)

	flag.IntVar(&nodeId, "nodeid", 55, "Node ID to use for client")
	flag.StringVar(&dumpFilename, "dumpfile", "", "Dump file to process")
	flag.StringVar(&intName, "interface", "", "CAN Network Interface name")
	flag.BoolVar(&captureAll, "capture", false, "Capture all CAN packets for debugging")
	flag.StringVar(&captureFn, "capture-filename", "output", "Capture filename [default: output]")
	flag.Parse()

	if dumpFilename == "" && intName == "" {
		fmt.Println("Nothing to do. Specify either a dump filename or interface name.")

		return
	}

	dev := zcan.NewZehnderDevice(byte(nodeId & 0xff))
	if intName != "" {
		if err := dev.Connect(intName); err != nil {
			fmt.Println(err)
			return
		}
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
		dest.GetMultiple(dev, []byte{4, 6, 8}, zcan.ZehnderRMITypeActualValue, showMultiple)

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
