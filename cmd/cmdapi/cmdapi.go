package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/mdm"
	mdmhttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/ctxlog"
	"github.com/micromdm/nanomdm/log/stdlogfmt"
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
		fmt.Println("must provide API keys")
		os.Exit(1)
	}

	logger := stdlogfmt.New(stdlog.Default(), true)

	var handler http.Handler = M2NCommandHandler(*flNanoURL, *flNanoKey, logger.With("handler", "command-handler"))
	handler = mdmhttp.BasicAuthMiddleware(handler, "micromdm", *flMicroKey, "micromdm")

	mux := http.NewServeMux()

	mux.Handle("/v1/commands", handler)
	mux.Handle("/version", mdmhttp.VersionHandler(version))

	rand.Seed(time.Now().UnixNano())

	logger.Info("msg", "starting server", "listen", *flListen)
	err := http.ListenAndServe(*flListen, mdmhttp.TraceLoggingMiddleware(mux, logger, newTraceID))
	if err != nil {
		logger.Info("msg", "server stopped", "err", err)
		os.Exit(1)
	}
}

func M2NCommandHandler(url string, key string, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		if r.Method != http.MethodPost {
			logger.Info("err", "POST method required")
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
			logger.Info(
				"msg", "new command",
				"udid", cmdReq.UDID,
				"request_type", cmdPayload.Command.RequestType,
				"command_uuid", cmdPayload.CommandUUID,
			)
			plistBytes, err = plist.Marshal(cmdPayload)
		} else {
			logger.Info("msg", "parsing body", "err", err)
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
			logger.Info("err", err)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(w).Encode(jsonResp)
		if err != nil {
			logger.Info("msg", "encoding JSON", "err", err)
		}
	}
}

// newTraceID generates a new HTTP trace ID for context logging.
// Currently this just makes a random string. This would be better
// served by e.g. https://github.com/oklog/ulid or something like
// https://opentelemetry.io/ someday.
func newTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
