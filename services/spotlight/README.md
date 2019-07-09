# Spotlight

Url image preview generator

## Running locally

### Start

```
make build-linux
cp ../../bin/spotlightd bin/spotlightd
cp example.env spotlightd.env  # Update as required
CURRENT_UID=$(id -u):$(id -g) docker-compose up --build -d spotlightd
```

### Test

Visit [http://localhost:1337/api/v1/spotlight?story_id=132](http://localhost:1337/api/v1/spotlight?story_id=132)


### Stop

```
docker-compose down
```

## Trouble shooting

### Packr2

```
$ make build
...
...
packr2 clean
make[1]: packr2: No such file or directory
make[1]: *** [deps] Error 1
make: *** [build] Error 2
```

Make sure $GOPATH/bin is linked in your $PATH

```
$PATH=$GOPATH/bin:$PATH
```

### Logging

```
docker ps
docker logs [Container ID obtained from docker ps]
```