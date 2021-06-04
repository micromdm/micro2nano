# micro2nano Operations Guide

`micro2nano` is a set of tools to assist in migrating from [MicroMDM](https://github.com/micromdm/micromdm) to [NanoMDM](https://github.com/micromdm/nanomdm).

## cmdapi

`cmdapi` is a simple server that actively translates MicroMDM's JSON-based command API to NanoMDM's raw Plist API. It listens on the listen address for "/v1/commands" endpoint.

### Switches

#### -api-key string

* MicroMDM API key

This is the *MicroMDM* pseudo-API key for authorizing incoming connections to this server.

#### -listen string

* listen address (default ":9001")

Specifies the listen address (interface & port number) for the server to listen on.

#### -nano-api-key string

* NanoMDM API key

This is the *NanoMDM* API key for authenticating to the NanoMDM instance.

#### -nano-url string

* NanoMDM Command URL

The URL of the NanoMDM command API endpoint. For example "http://127.0.0.1:9000/v1/enqueue".

## llorne

`llorne` (enroll spelled backwards) is a tool that works with NanoMDM's "migration" endpoint to import MicroMDM enrollment data into NanoMDM from a MicroMDM database.

### Switches

#### -db string

* path to micromdm DB (default "/var/db/micromdm.db")

#### -key string

* NanoMDM API Key

This is the NanoMDM API key for authenticating to the NanoMDM server.

#### -url string

The URL of the NanoMDM migration endpoint. For example "http://127.0.0.1:9000/migration".
