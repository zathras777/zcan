package zcan

func (dev *ZehnderDevice) processFrame() {
loop:
	for {
		select {
		case frame := <-dev.frameQ:
			ck := frame.ID >> 24
			switch ck {
			case 0:
				dev.pdoQ <- frame
			case 0x1F:
				dev.rmiQ <- frame
			case 0x10:
				dev.heartbeatQ <- frame
			}
		case <-dev.stopSignal:
			break loop
		}
	}
}
