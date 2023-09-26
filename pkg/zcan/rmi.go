package zcan

import (
	"fmt"

	"go.einride.tech/can"
)

func (dev *ZehnderDevice) processRMIFrame() {
	var holder *zehnderRMI
loop:
	for {
		select {
		case frame := <-dev.rmiQ:
			rmi := rmiFromFrame(frame)
			if rmi.DestId != dev.NodeID {
				if rmi.SourceId == dev.NodeID {
					continue
				}
				fmt.Printf("Received RMI but it's for us...%02X vs wanted %02X\n", rmi.DestId, dev.NodeID)
				fmt.Println(rmi)
				continue
			}
			if rmi.IsMulti {
				if holder != nil {
					holder.appendRMI(rmi)
				} else {
					holder = rmi
				}
				if holder.finalSeen {
					dev.doRMICallback(holder)
					holder = nil
				}
			} else {
				dev.doRMICallback(rmi)
			}
		case <-dev.stopSignal:
			break loop
		}
	}
}

type ZehnderTypeFlag byte

const (
	ZehnderRMITypeNoValue     ZehnderTypeFlag = 0x00
	ZehnderRMITypeActualValue ZehnderTypeFlag = 0x10
	ZehnderRMITypeRange       ZehnderTypeFlag = 0x20
	ZehnderRMITypeStepSize    ZehnderTypeFlag = 0x40
)

type zehnderRMI struct {
	SourceId   byte
	DestId     byte
	Sequence   byte
	Counter    byte
	IsMulti    bool
	IsRequest  bool
	IsError    bool
	Data       []byte
	DataLength int

	msgNo      byte
	finalSeen  bool
	callbackFn func([]byte)
}

type ZehnderDestination struct {
	DestNodeId byte
	Unit       byte
	SubUnit    byte
}

func NewZehnderDestination(node byte, unit byte, subunit byte) ZehnderDestination {
	return ZehnderDestination{node, unit, subunit}
}

func (zr ZehnderDestination) GetOne(dev *ZehnderDevice, prop byte, flags ZehnderTypeFlag, cbFn func([]byte)) {
	rmi := zehnderRMI{SourceId: dev.NodeID, DestId: zr.DestNodeId, IsRequest: true, Sequence: dev.rmiSequence}
	rmi.Data = []byte{0x01, zr.Unit, zr.SubUnit, byte(flags), prop}
	rmi.DataLength = 5
	rmi.callbackFn = cbFn
	dev.rmiSequence = (dev.rmiSequence + 1) & 0x03
	dev.rmiRequestQ <- &rmi
}

func (zr ZehnderDestination) GetMultiple(dev *ZehnderDevice, props []byte, flags ZehnderTypeFlag, cbFn func([]byte)) {
	rmi := zehnderRMI{SourceId: dev.NodeID, DestId: zr.DestNodeId, IsRequest: true, Sequence: dev.rmiSequence}
	or_type := byte(flags) | byte(len(props))
	rmi.Data = append([]byte{0x02, zr.Unit, zr.SubUnit, 1, or_type}, props...)
	rmi.DataLength = len(rmi.Data)
	if rmi.DataLength > 8 {
		rmi.IsMulti = true
	}
	rmi.callbackFn = cbFn
	dev.rmiSequence = (dev.rmiSequence + 1) & 0x03
	dev.rmiRequestQ <- &rmi
}

func rmiFromFrame(frame can.Frame) *zehnderRMI {
	rmi := zehnderRMI{SourceId: byte(frame.ID & 0x3F)}
	rmi.DestId = byte(frame.ID>>6) & 0x3F
	rmi.Counter = byte(frame.ID>>12) & 0x03
	rmi.Sequence = byte(frame.ID>>17) & 0x03
	rmi.IsMulti = frame.ID&1<<14 == 1<<14
	rmi.IsError = frame.ID&1<<15 == 1<<15
	rmi.IsRequest = frame.ID&1<<16 == 1<<16
	rmi.Data = frame.Data[:frame.Length]
	rmi.DataLength = int(frame.Length)

	if !rmi.IsMulti {
		rmi.finalSeen = true
	} else {
		rmi.msgNo = rmi.Data[0]
		rmi.DataLength -= 1
		rmi.Data = rmi.Data[1:]
	}
	return &rmi
}

func (zrmi *zehnderRMI) appendRMI(xtra *zehnderRMI) {
	zrmi.msgNo = xtra.msgNo
	if zrmi.msgNo&0x80 == 0x80 {
		zrmi.finalSeen = true
		zrmi.msgNo &= 0x7F
	}
	zrmi.Data = append(zrmi.Data, xtra.Data...)
	zrmi.DataLength += xtra.DataLength
}

func (zrmi zehnderRMI) makeCANId() uint32 {
	can_id := uint32(0x1F000000) + uint32(zrmi.SourceId)
	can_id += uint32(zrmi.DestId << 6)
	can_id += uint32(zrmi.Counter&0x03) << 12
	if zrmi.IsMulti {
		can_id += (1 << 14)
	}
	if zrmi.IsRequest {
		can_id += (1 << 16)
	}
	can_id += uint32(zrmi.Sequence&0x03) << 17
	return can_id
}

func (zrmi *zehnderRMI) send(dev *ZehnderDevice) error {
	dev.rmiCbFn = zrmi.callbackFn
	frame := can.Frame{ID: zrmi.makeCANId(), IsExtended: true}
	copy(frame.Data[:], zrmi.Data[:])
	frame.Length = uint8(zrmi.DataLength)
	dev.txQ <- frame
	return nil
}

func (dev *ZehnderDevice) doRMICallback(rmi *zehnderRMI) {
	if dev.rmiCbFn != nil {
		dev.rmiCbFn(rmi.Data[:rmi.DataLength])
		dev.rmiCbFn = nil
	} else {
		fmt.Println("RMI message received, but no callback was set?")
	}
	dev.rmiCTS <- true
}

func (dev *ZehnderDevice) processRMIQueue() {
	dev.wg.Add(1)
loop:
	for {
		select {
		case <-dev.rmiCTS:
			select {
			case rmi := <-dev.rmiRequestQ:
				rmi.send(dev)
			case <-dev.stopSignal:
				break loop
			}
		case <-dev.stopSignal:
			break loop
		}
	}
	dev.wg.Done()
}
