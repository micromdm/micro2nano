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

#### -version

* print version

Print version and exit.

## llorne

`llorne` (enroll spelled backwards) is a tool that works with NanoMDM's "migration" endpoint to import MicroMDM enrollment data into NanoMDM from a MicroMDM database.

### Switches

#### -days int

* Skip processing devices with a last seen older than this many days

Skip processing of any device with a Last Seen date older than this many days. This feature originally intended to cull very old/dated enrollments from the database.

#### -db string

* path to micromdm DB (default "/var/db/micromdm.db")

#### -key string

* NanoMDM API Key

This is the NanoMDM API key for authenticating to the NanoMDM server.

#### track-path string

* Path to tracking database to avoid sending duplicate messages

Path to a separate BoltDB database that keeps track of send Authenticate and TokenUpdate messages and prevents re-sending the same message twice. We track "seen" and "sent" messages by the hash of their contents as MicroMDM does not have the ability to track a device's enrollment date.

#### -url string

The URL of the NanoMDM migration endpoint. For example "http://127.0.0.1:9000/migration".

#### -version

* print version

Print version and exit.
