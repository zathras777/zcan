package zcan

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	var data []string
	for _, element := range dev.pdoData {
		data = append(data, element.jsonString())
	}
	io.WriteString(w, "{\"name\": \""+dev.Name+"\", "+strings.Join(data, ", ")+"}")
}

func (dev *ZehnderDevice) dumpPDO(w http.ResponseWriter, r *http.Request) {
	dev.DumpPDO()
}
