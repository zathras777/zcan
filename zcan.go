package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zathras777/zcan/pkg/zcan"
)

func showSerialNumber(data []byte) {
	fmt.Println(string(data))
}

func showMultiple(data []byte) {
	fmt.Println(data)
}

func main() {
	var (
		nodeId       int
		dumpFilename string
		intName      string
	)

	flag.IntVar(&nodeId, "nodeid", 55, "Node ID to use for client")
	flag.StringVar(&dumpFilename, "dumpfile", "", "Dump file to process")
	flag.StringVar(&intName, "interface", "", "CAN Network Interface name")
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

	dev.Start()

	if dumpFilename != "" {
		fmt.Printf("Processing dumpfile: %s\n", dumpFilename)
		dev.ProcessDumpFile(dumpFilename)
		dev.Stop()
	} else {

		dest := zcan.NewZehnderDestination(1, 1, 1)
		dest.GetOne(dev, 4, zcan.ZehnderRMITypeActualValue, showSerialNumber)
		dest.GetOne(dev, 8, zcan.ZehnderRMITypeActualValue, showSerialNumber)
		dest.GetMultiple(dev, []byte{4, 6, 8}, zcan.ZehnderRMITypeActualValue, showMultiple)

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigs
			dev.Stop()
		}()
	}

	fmt.Println("Processing CAN packets. CTRL+C to quit...")

	dev.Wait()

	dev.DumpPDO()
}
