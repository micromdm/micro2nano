package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	"github.com/groob/plist"
	apnsbuiltin "github.com/micromdm/micromdm/platform/apns/builtin"
	"github.com/micromdm/micromdm/platform/device"
	devicebuiltin "github.com/micromdm/micromdm/platform/device/builtin"
	"github.com/micromdm/micromdm/platform/pubsub/inmem"
	userbuiltin "github.com/micromdm/micromdm/platform/user/builtin"
)

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
		flDB = flag.String("db", "/var/db/micromdm.db", "path to micromdm DB")
	)
	flag.Parse()
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
		// TODO:
		if pushInfo.PushMagic == "fakePushMagic" {
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
		fmt.Println(string(authPlist))
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
		fmt.Println(string(tokenPlist))
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
		fmt.Println("user", user.UserID)
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
		fmt.Println(string(tokenPlist))
	}

	return
}
