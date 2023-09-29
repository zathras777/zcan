package zcan

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"go.einride.tech/can"
)

func slugify(orig string) string {
	slug := strings.ToLower(orig)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}

func (dev *ZehnderDevice) processPDOFrame() {
loop:
	for {
		select {
		case frame := <-dev.pdoQ:
			msg := pdoFromFrame(frame)
			if msg.pdoId == 0 {
				fmt.Println("Ignoring PDO with an ID of 0")
				continue
			}
			//fmt.Println(msg)
			pv, ck := dev.pdoData[int(msg.pdoId)]
			if !ck {
				sensor := findSensor(int(msg.pdoId), msg.length)
				pv = &PDOValue{sensor, nil, slugify(sensor.Name)}
				dev.pdoData[int(msg.pdoId)] = pv
			}
			pv.Value = msg.data[:msg.length]
		case <-dev.stopSignal:
			break loop
		}
	}
}

type pdoMessage struct {
	nodeId uint32
	pdoId  uint32
	length int
	data   []byte
}

func pdoFromFrame(frame can.Frame) pdoMessage {
	return pdoMessage{nodeId: frame.ID & 0x3F, pdoId: (frame.ID >> 14) & 0x7FF, length: int(frame.Length), data: frame.Data[:frame.Length]}
}

func (pdo pdoMessage) String() string {
	return fmt.Sprintf("Node ID: %d, PDO ID: %d  => 0x%s", pdo.nodeId, pdo.pdoId, strings.ToUpper(hex.EncodeToString(pdo.data[:pdo.length])))
}

const (
	UNIT_WATT    = "W"
	UNIT_KWH     = "kWh"
	UNIT_CELCIUS = "°C"
	UNIT_PERCENT = "%"
	UNIT_RPM     = "rpm"
	UNIT_M3H     = "m³/h"
	UNIT_SECONDS = "seconds"
	UNIT_UNKNOWN = "unknown"
	UNIT_DAYS    = "Days"
)

type ZehnderType int

const (
	CN_BOOL   ZehnderType = iota // 00 (false), 01 (true)
	CN_UINT8                     // 00 (0) until ff (255)
	CN_UINT16                    // 3412 = 1234
	CN_UINT32                    // 7856 3412 = 12345678
	CN_INT8
	CN_INT16 //3412 = 1234
	CN_INT64
	CN_STRING
	CN_TIME
	CN_VERSION
)

type PDOSensor struct {
	Name          string
	Units         string
	DataType      ZehnderType
	DecimalPlaces int
}

type PDOValue struct {
	Sensor PDOSensor
	Value  []byte
	slug   string
}

var sensorData = map[int]PDOSensor{
	49:  {"Operating Mode", UNIT_UNKNOWN, CN_INT8, 0},
	65:  {"Fan Speed Setting", UNIT_UNKNOWN, CN_INT8, 0},
	81:  {"Boost Period Remaining", UNIT_SECONDS, CN_UINT32, 0},
	117: {"Exhaust Fan Duty", UNIT_PERCENT, CN_UINT8, 0},
	118: {"Supply Fan Duty", UNIT_PERCENT, CN_UINT8, 0},
	119: {"Exhaust Fan Flow", UNIT_M3H, CN_UINT16, 0},
	120: {"Supply Fan Flow", UNIT_M3H, CN_UINT16, 0},
	121: {"Exhaust Fan Speed", UNIT_RPM, CN_UINT16, 0},
	122: {"Supply Fan Speed", UNIT_RPM, CN_UINT16, 0},
	128: {"Power Consumption", UNIT_WATT, CN_UINT16, 0},
	130: {"Power Consumption Total", UNIT_KWH, CN_UINT16, 0},
	145: {"Preheater Power Consumption Total", UNIT_KWH, CN_UINT16, 0},
	146: {"Preheater Power Consumption", UNIT_WATT, CN_UINT16, 0},
	192: {"Filter Replacement Days", UNIT_DAYS, CN_UINT16, 0},
	209: {"RMOT", UNIT_CELCIUS, CN_UINT16, 1},
	213: {"Avoided Heating Actual", UNIT_WATT, CN_UINT16, 2},
	214: {"Avoided Heating YTD", UNIT_KWH, CN_UINT16, 0},
	220: {"Preheated Air Temperature (pre Heating)", UNIT_CELCIUS, CN_UINT16, 1},
	221: {"Preheated Air Temperature (post Heating)", UNIT_CELCIUS, CN_UINT16, 1},
	227: {"Bypass State", UNIT_PERCENT, CN_UINT8, 0},
	274: {"Extract Air Temperature", UNIT_CELCIUS, CN_UINT16, 1},
	275: {"Exhaust Air Temperature", UNIT_CELCIUS, CN_UINT16, 1},
	276: {"Outdoor Air Temperature", UNIT_CELCIUS, CN_UINT16, 1},
	277: {"Preheated Outside Air Temperature", UNIT_CELCIUS, CN_UINT16, 1},
	278: {"Supply Air Temperature", UNIT_CELCIUS, CN_UINT16, 1},
	290: {"Extract Humidity", UNIT_PERCENT, CN_UINT8, 0},
	291: {"Exhaust Humidity", UNIT_PERCENT, CN_UINT8, 0},
	292: {"Outdoor Humidity", UNIT_PERCENT, CN_UINT8, 0},
	293: {"Preheated Outdoor Humidity", UNIT_PERCENT, CN_UINT8, 0},
	294: {"Supply Air Humidity", UNIT_PERCENT, CN_INT8, 0},
}

func findSensor(pdo int, dataLen int) PDOSensor {
	sensor, ck := sensorData[pdo]
	if !ck {
		sensor = PDOSensor{fmt.Sprintf("Unknown sensor %d", pdo), UNIT_UNKNOWN, CN_UINT16, 0}
		if dataLen == 1 {
			sensor.DataType = CN_UINT8
		} else if dataLen == 4 {
			sensor.DataType = CN_UINT32
		}
		sensorData[pdo] = sensor
	}
	return sensor
}

func (pv PDOValue) String() string {
	s := fmt.Sprintf("%-45s0x%-8s", pv.Sensor.Name, strings.ToUpper(hex.EncodeToString(pv.Value)))
	if pv.IsFloat() {
		fmtS := fmt.Sprintf("  %%6.%df", pv.Sensor.DecimalPlaces)
		s += fmt.Sprintf(fmtS, pv.Float())
	} else if pv.IsSigned() {
		s += fmt.Sprintf("  %6d", pv.SignedNumber())
	} else {
		s += fmt.Sprintf("  %6d", pv.Number())
	}
	s += " " + pv.Sensor.Units
	return s
}

func (pv PDOValue) jsonString() string {
	var val string
	if pv.IsFloat() {
		fmtS := fmt.Sprintf("%%.%df", pv.Sensor.DecimalPlaces)
		val = fmt.Sprintf(fmtS, pv.Float())
	} else if pv.IsSigned() {
		val = fmt.Sprintf("%d", pv.SignedNumber())
	} else {
		val = fmt.Sprintf("%d", pv.Number())
	}
	return fmt.Sprintf("\"%s\": %s", pv.slug, val)
}

func (pv PDOValue) IsBool() bool   { return pv.Sensor.DataType == CN_BOOL }
func (pv PDOValue) IsString() bool { return pv.Sensor.DataType == CN_STRING }
func (pv PDOValue) IsFloat() bool  { return pv.Sensor.DecimalPlaces > 0 }
func (pv PDOValue) IsSigned() bool {
	return pv.Sensor.DataType == CN_INT8 || pv.Sensor.DataType == CN_INT16 || pv.Sensor.DataType == CN_INT64
}

func (pv PDOValue) Number() uint {
	if pv.Sensor.DataType == CN_INT16 || pv.Sensor.DataType == CN_INT8 || pv.Sensor.DataType == CN_INT64 {
		log.Println("attempt to get an unsigned number from a sensor with a signed data type?")
		return 0
	}
	switch pv.Sensor.DataType {
	case CN_UINT8:
		return uint(pv.Value[0])
	case CN_UINT16:
		return uint(binary.LittleEndian.Uint16(pv.Value))
	case CN_UINT32:
		return uint(binary.LittleEndian.Uint32(pv.Value))
	}
	return 0
}

func (pv PDOValue) SignedNumber() int {
	if pv.Sensor.DataType == CN_UINT16 || pv.Sensor.DataType == CN_UINT8 || pv.Sensor.DataType == CN_UINT32 {
		log.Println("attempt to get an signed number from a sensor with an unsigned data type?")
		return 0
	}
	switch pv.Sensor.DataType {
	case CN_INT8:
		return int(pv.Value[0])
	case CN_INT16:
		return int(binary.LittleEndian.Uint16(pv.Value))
	case CN_INT64:
		return int(binary.LittleEndian.Uint32(pv.Value))
	}
	return 0
}

func (pv PDOValue) Float() float64 {
	return float64(pv.Number()) / (float64(pv.Sensor.DecimalPlaces) * 10)
}
