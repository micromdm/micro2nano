package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/mdm"
)

func main() {
	var (
		flListen   = flag.String("listen", ":9001", "listen address")
		flMicroKey = flag.String("api-key", "", "MicroMDM API key")
		flNanoKey  = flag.String("nano-api-key", "", "NanoMDM API key")
		flNanoURL  = flag.String("nano-url", "", "NanoMDM Command URL")
	)
	flag.Parse()

	if *flMicroKey == "" || *flNanoKey == "" {
		log.Fatal("must provide API keys")
	}

	var handler http.Handler = M2NCommandHandler(*flNanoURL, *flNanoKey)
	handler = basicAuth(handler, "micromdm", *flMicroKey, "micromdm")
	handler = simpleLog(handler)

	http.Handle("/v1/commands", handler)

	log.Printf("starting server %s\n", *flListen)
	http.ListenAndServe(*flListen, nil)
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
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		cmdPayload, err := mdm.NewCommandPayload(cmdReq)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		log.Printf("new command: udid=%s request_type=%s uuid=%s\n", cmdReq.UDID, cmdPayload.Command.RequestType, cmdPayload.CommandUUID)
		plistBytes, err := plist.Marshal(cmdPayload)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), "GET", url+"/"+cmdReq.UDID, bytes.NewBuffer(plistBytes))
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		req.SetBasicAuth("nanomdm", key)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		responseData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

		w.Write(responseData)
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