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
	dataMap["model"] = dev.Name
	dataMap["serial_number"] = dev.SerialNumber

	for _, v := range dev.pdoData {
		dataMap[v.Sensor.slug] = v.GetData()
	}

	outData, err := json.Marshal(dataMap)
	if err == nil {
		w.Write(outData)
		return
	}
	log.Printf("Unable to generate json data: %s", err)
}

func (dev *ZehnderDevice) dumpPDO(w http.ResponseWriter, r *http.Request) {
	dev.DumpPDO()
}
