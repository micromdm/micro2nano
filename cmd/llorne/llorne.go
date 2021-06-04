package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/boltdb/bolt"
	"github.com/groob/plist"
	apnsbuiltin "github.com/micromdm/micromdm/platform/apns/builtin"
	"github.com/micromdm/micromdm/platform/device"
	devicebuiltin "github.com/micromdm/micromdm/platform/device/builtin"
	"github.com/micromdm/micromdm/platform/pubsub/inmem"
	userbuiltin "github.com/micromdm/micromdm/platform/user/builtin"
)

// overridden by -ldflags -X
var version = "unknown"

type Authenticate struct {
	MessageType  string
	UDID         string
	Topic        string
	BuildVersion string `plist:",omitempty"`
	DeviceName   string `plist:",omitempty"`
	Model        string `plist:",omitempty"`
	ModelName    string `plist:",omitempty"`
	OSVersion    string `plist:",omitempty"`
	ProductName  string `plist:",omitempty"`
	SerialNumber string `plist:",omitempty"`
	IMEI         string `plist:",omitempty"`
	MEID         string `plist:",omitempty"`
}

type TokenUpdate struct {
	MessageType   string
	UDID          string
	PushMagic     string
	Topic         string
	Token         []byte
	UnlockToken   []byte `plist:",omitempty"`
	UserID        string `plist:",omitempty"`
	UserShortName string `plist:",omitempty"`
	UserLongName  string `plist:",omitempty"`
}

func main() {
	var (
		flDB      = flag.String("db", "/var/db/micromdm.db", "path to micromdm DB")
		flURL     = flag.String("url", "", "NanoMDM migration URL")
		flKey     = flag.String("key", "", "NanoMDM API Key")
		flVersion = flag.Bool("version", false, "print version")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	var skipServer bool
	if *flURL == "" || *flKey == "" {
		log.Println("URL or API key not set; not sending server requests")
		skipServer = true
	}
	client := http.DefaultClient
	if _, err := os.Stat(*flDB); err != nil {
		log.Fatal(err)
	}
	boltDB, err := bolt.Open(*flDB, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	ps := inmem.NewPubSub()

	apnsDB, err := apnsbuiltin.NewDB(boltDB, ps)
	if err != nil {
		log.Fatal(err)
	}

	deviceDB, err := devicebuiltin.NewDB(boltDB)
	if err != nil {
		log.Fatal(err)
	}
	devices, err := deviceDB.List(context.Background(), device.ListDevicesOption{})
	if err != nil {
		log.Fatal(err)
	}
	for _, device := range devices {
		pushInfo, err := apnsDB.PushInfo(context.Background(), device.UDID)
		if err != nil {
			log.Println(err)
			continue
		}
		authenticate := &Authenticate{
			MessageType: "Authenticate",
			UDID:        device.UDID,

			Topic:        pushInfo.MDMTopic,
			BuildVersion: device.BuildVersion,
			DeviceName:   device.DeviceName,
			Model:        device.Model,
			ModelName:    device.ModelName,
			OSVersion:    device.OSVersion,
			ProductName:  device.ProductName,
			SerialNumber: device.SerialNumber,
			IMEI:         device.IMEI,
			MEID:         device.MEID,
		}
		authPlist, err := plist.Marshal(authenticate)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Printf("sending device Authenticate for: UDID=%s\n", authenticate.UDID)
		if !skipServer {
			if err := put(client, *flURL, *flKey, authPlist); err != nil {
				log.Println(err)
				continue
			}
		}
		token, err := hex.DecodeString(pushInfo.Token)
		if err != nil {
			log.Println(err)
			continue
		}
		unlockToken, err := hex.DecodeString(device.UnlockToken)
		if err != nil {
			log.Println(err)
			continue
		}
		tokenUpdate := &TokenUpdate{
			MessageType: "TokenUpdate",
			UDID:        device.UDID,

			PushMagic: pushInfo.PushMagic,
			Token:     token,
			Topic:     pushInfo.MDMTopic,

			UnlockToken: unlockToken,
		}
		tokenPlist, err := plist.Marshal(tokenUpdate)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Printf("sending device TokenUpdate for: UDID=%s\n", tokenUpdate.UDID)
		if !skipServer {
			if err := put(client, *flURL, *flKey, tokenPlist); err != nil {
				log.Println(err)
				continue
			}
		}
	}

	userDB, err := userbuiltin.NewDB(boltDB)
	if err != nil {
		log.Fatal(err)
	}
	users, err := userDB.List()
	if err != nil {
		log.Fatal(err)
	}
	for _, user := range users {
		pushInfo, err := apnsDB.PushInfo(context.Background(), user.UserID)
		if err != nil {
			log.Println(err)
			continue
		}
		token, err := hex.DecodeString(pushInfo.Token)
		if err != nil {
			log.Println(err)
			continue
		}
		tokenUpdate := &TokenUpdate{
			MessageType: "TokenUpdate",
			UDID:        user.UDID,
			UserID:      user.UserID,

			PushMagic: pushInfo.PushMagic,
			Token:     token,
			Topic:     pushInfo.MDMTopic,

			UserShortName: user.UserShortname,
			UserLongName:  user.UserLongname,
		}
		tokenPlist, err := plist.Marshal(tokenUpdate)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Printf("sending user TokenUpdate for: UserID=%s UserShortName=%s\n", tokenUpdate.UserID, tokenUpdate.UserShortName)
		if !skipServer {
			if err := put(client, *flURL, *flKey, tokenPlist); err != nil {
				log.Println(err)
				continue
			}
		}
	}
	return
}

func put(client *http.Client, url string, key string, sendBytes []byte) error {
	if url == "" || key == "" {
		return errors.New("no URL or API key set")
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(sendBytes))
	if err != nil {
		return err
	}
	req.SetBasicAuth("nanomdm", key)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Check-in Request failed with HTTP status: %d", res.StatusCode)
	}
	return nil
}
