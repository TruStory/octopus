# Push Notification Service

## Local Setup

Clone this repo to a location _**outside**_ of `GOPATH`, since the new dependency manager (go modules) is made to work outside of `GOPATH`.

### Environment variables

```
cp example.gorush.env gorush.env
cp example.pushd.env pushd.env
```

#### go-rush

Download the required certificates from the certificates repo: https://github.com/TruStory/certificates/tree/master/apns.

Copy the development cert into a location such as `./certs` and set `GORUSH_IOS_KEY_PATH`.

`GORUSH_IOS_PASSWORD` is the value from [pwd.tx](https://github.com/TruStory/certificates/blob/master/apns/pwd.tx).

```
GORUSH_IOS_ENABLED=true
GORUSH_IOS_KEY_TYPE=p12
GORUSH_IOS_KEY_PATH=./certs/development_io.trustory.app.devnet.p12
GORUSH_IOS_PASSWORD=password
GORUSH_IOS_PRODUCTION=false
```

#### pushd

This is the exact same DB used for TruChain with the same credentials.

```
PG_ADDR=dbaddress:5432
PG_USER=dbuser
PG_USER_PW=dbpwd
PG_DB_NAME=trudb
REMOTE_ENDPOINT=tcp://127.0.0.1:26657
```

### Running

Via Go: `make run`

Via Docker: `make run-docker`