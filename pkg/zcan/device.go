package zcan

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"go.einride.tech/can"
)

func ZehnderVersionDecode(val uint32) (major int, minor int) {
	major = int(val>>30) & 3
	minor = int(val>>20) & 1023
	return
}

type ZehnderDevice struct {
	NodeID    byte
	Connected bool

	connection zConnection

	wg          sync.WaitGroup
	routines    int
	stopSignal  chan bool
	frameQ      chan can.Frame
	pdoQ        chan can.Frame
	rmiQ        chan can.Frame
	txQ         chan can.Frame
	heartbeatQ  chan can.Frame
	rmiRequestQ chan *zehnderRMI
	rmiCTS      chan bool
	pdoData     map[int]*PDOValue
	rmiCbFn     func([]byte)
	rmiSequence byte
	captureFh   *os.File
	doCapture   bool
}

func NewZehnderDevice(id byte) *ZehnderDevice {
	return &ZehnderDevice{NodeID: id, pdoData: make(map[int]*PDOValue)}
}

func (dev *ZehnderDevice) Connect(interfaceName string) error {
	return dev.connection.open_device(interfaceName)
}

func (dev *ZehnderDevice) Start() error {
	dev.stopSignal = make(chan bool, 2)
	dev.frameQ = make(chan can.Frame)
	dev.pdoQ = make(chan can.Frame)
	dev.rmiQ = make(chan can.Frame)
	dev.txQ = make(chan can.Frame)
	dev.heartbeatQ = make(chan can.Frame)
	dev.rmiRequestQ = make(chan *zehnderRMI)
	dev.rmiCTS = make(chan bool, 1)

	go dev.processFrame()
	go dev.processPDOFrame()
	go dev.processRMIFrame()
	go dev.processRMIQueue()
	dev.routines = 4

	if dev.connection.device != nil {
		// The receiver does not participate in the wait group, so
		// don't include in the numbers...
		go dev.receiver()
		go dev.transmitter()
		go dev.heartbeat()
		dev.rmiCTS <- true
		dev.routines = 6
	}

	return nil
}

func (dev *ZehnderDevice) Wait() {
	dev.wg.Wait()
}

func (dev *ZehnderDevice) Stop() {
	for n := 0; n < dev.routines; n++ {
		dev.stopSignal <- true
	}
}

func (dev *ZehnderDevice) CaptureAll(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		fmt.Println(err)
		return err
	}
	dev.captureFh = f
	dev.doCapture = true
	return nil
}

func (dev *ZehnderDevice) ProcessDumpFile(filename string) (err error) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s. Error: %s\n", filename, err)
		return err
	}
	fmt.Printf("File: %s. Total size is %v bytes\n", filename, info.Size())
	if info.Size() == 0 {
		fmt.Println("File has 0 bytes. Nothing to do")
		return fmt.Errorf("file has zero size. Nothing to do")
	}

	readFile, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
		return err
	}
	fileScanner := bufio.NewScanner(readFile)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		frame := can.Frame{}
		frame.UnmarshalString(fileScanner.Text())
		dev.frameQ <- frame
	}

	readFile.Close()
	return err
}

func (dev *ZehnderDevice) DumpPDO() {
	fmt.Println()
	for key, element := range dev.pdoData {
		fmt.Printf("%3d: %s\n", key, element)
	}
}
