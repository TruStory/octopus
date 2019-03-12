# TruStory Uploader Service

## Environment setup

```
Install Go
```

## CORS

In order to speed deployment and configuration, the url generating service
is open to the world using CORS and `Access-Control-Allow-Origin: *`.

## Run

Create `config.toml` and fill in required data

```
AWSKey=[your key]
AWSSecret=[your secret]
Port="4000"
BucketName="trustory"
Region="us-west-1"
ImageFolder="images/"
```

```
go build -o uploader app.go
./uploader
```
