package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/mdm"
)

// overridden by -ldflags -X
var version = "unknown"

func main() {
	var (
		flListen   = flag.String("listen", ":9001", "listen address")
		flMicroKey = flag.String("api-key", "", "MicroMDM API key")
		flNanoKey  = flag.String("nano-api-key", "", "NanoMDM API key")
		flNanoURL  = flag.String("nano-url", "", "NanoMDM Command URL")
		flVersion  = flag.Bool("version", false, "print version")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *flMicroKey == "" || *flNanoKey == "" {
		log.Fatal("must provide API keys")
	}

	var handler http.Handler = M2NCommandHandler(*flNanoURL, *flNanoKey)
	handler = basicAuth(handler, "micromdm", *flMicroKey, "micromdm")
	handler = simpleLog(handler)

	http.Handle("/v1/commands", handler)
	http.Handle("/version", versionHandler(version))

	log.Printf("starting server %s\n", *flListen)
	http.ListenAndServe(*flListen, nil)
}

// versionHandler returns a simple JSON response from a version string.
func versionHandler(version string) http.HandlerFunc {
	bodyBytes := []byte(`{"version":"` + version + `"}`)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bodyBytes)
	}
}

func M2NCommandHandler(url string, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Println("POST method required")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		cmdReq := &mdm.CommandRequest{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&cmdReq)
		var cmdPayload *mdm.CommandPayload
		if err == nil {
			cmdPayload, err = mdm.NewCommandPayload(cmdReq)
		}
		var plistBytes []byte
		if err == nil {
			log.Printf("new command: udid=%s request_type=%s uuid=%s\n", cmdReq.UDID, cmdPayload.Command.RequestType, cmdPayload.CommandUUID)
			plistBytes, err = plist.Marshal(cmdPayload)
		} else {
			log.Printf("error parsing body: %v\n", err)
		}
		var req *http.Request
		if err == nil {
			req, err = http.NewRequestWithContext(r.Context(), "GET", url+"/"+cmdReq.UDID, bytes.NewBuffer(plistBytes))
		}
		var resp *http.Response
		if err == nil {
			req.SetBasicAuth("nanomdm", key)
			resp, err = http.DefaultClient.Do(req)
		}
		if err == nil {
			defer resp.Body.Close()
			_, err = ioutil.ReadAll(resp.Body)
		}
		jsonResp := &struct {
			Payload *mdm.CommandPayload `json:"payload,omitempty"`
			Err     string              `json:"error,omitempty"`
		}{}
		if err != nil {
			jsonResp.Err = err.Error()
		} else {
			jsonResp.Payload = cmdPayload
		}
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(w).Encode(jsonResp)
		if err != nil {
			log.Println(err)
		}
	}
}

func basicAuth(next http.Handler, username, password, realm string) http.HandlerFunc {
	uBytes := []byte(username)
	pBytes := []byte(password)
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), uBytes) != 1 || subtle.ConstantTimeCompare([]byte(p), pBytes) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func simpleLog(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())
		next.ServeHTTP(w, r)
	}
}
