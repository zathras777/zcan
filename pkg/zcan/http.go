package zcan

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (dev *ZehnderDevice) startHttpServer(host string, port int) {
	log.Printf("Starting HTTP server listening @ http://%s:%d/", host, port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", dev.jsonResponse)
	mux.HandleFunc("/device-info", dev.jsonDeviceInfo)
	mux.HandleFunc("/dump", dev.dumpPDO)

	dev.http = &http.Server{Addr: fmt.Sprintf("%s:%d", host, port), Handler: mux}
	dev.wg.Add(1)
	err := dev.http.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Println("HTTP server shutdown")
		dev.wg.Done()
	} else if err != nil {
		log.Printf("server error: %v\n", err)
	}
}

func (dev *ZehnderDevice) jsonResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dataMap := make(map[string]interface{})

	for _, v := range dev.pdoData {
		dataMap[v.Sensor.slug] = v.GetData()
	}

	outData, err := json.Marshal(dataMap)
	if err == nil {
		w.Write(outData)
		return
	}
	log.Printf("jsonResponse: Unable to generate json data: %s", err)
}

func (dev *ZehnderDevice) jsonDeviceInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("dev.SerialNumber: %s", dev.SerialNumber)
	if dev.SerialNumber == "" {
		syncer := make(chan bool)
		dev.getDeviceInfo(syncer)
		<-syncer
	}
	dataMap := make(map[string]interface{})
	dataMap["model"] = dev.Model
	dataMap["serial_number"] = dev.SerialNumber
	dataMap["software_version"] = dev.SoftwareVersion

	outData, err := json.Marshal(dataMap)
	if err == nil {
		w.Write(outData)
		return
	}
	log.Printf("jsonDeviceInfo: Unable to generate json data: %s", err)
}

func (dev *ZehnderDevice) dumpPDO(w http.ResponseWriter, r *http.Request) {
	dev.DumpPDO()
}
