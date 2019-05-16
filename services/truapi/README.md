# TruStory API server

TruAPI serves as a web server for TruStory as well as a light client for the TruChain blockchain.

## Running

```
truapid start
```

## Home folder (.octopus)

Contains:
* config.toml file (see below)
* local key store
* Twitter whitelist

## Configuration

Config vars can be set in 3 ways:

i.e: for the parameter "app.name":
1. Command-line flag: `--app.name TruStory`
2. Env var: `APP_NAME=TruStory`
3. config.toml in .truapid/config
```toml
[app] 
name = TruStory
```
4. Default value "TruStory" if not supplied by the above

Precedence: 1 -> 2 -> 3 -> 4
