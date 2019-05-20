# TruStory API server

TruAPI serves as an HTTP server for the TruStory mobile and web apps, as well as a light client for the TruChain blockchain.

## Home folder (.octopus)

Contains:
* config.toml file (see below)
* local key store
* Twitter whitelist

## Environment configuration

Config vars can be set in 3 ways:

i.e: for the parameter "app.name":
1. Command-line flag: `--app.name=TruStory`
2. Env var: `APP_NAME=TruStory`
3. config.toml in .truapid/config
```toml
[app] 
name = TruStory
```
4. Default value "TruStory" if not supplied by the above

Precedence: 1 -> 2 -> 3 -> 4

## Running

```
# Build the binary
make build

# Start the light client
./bin/truapid start --home ~/.octopus --chain-id truchain
```

## GraphQL Queries
You can reach your client at `http://localhost:1337/api/v1/graphql/`

Sample query:
```graphql
{
  stories {
    id
    body
    backings {
      amount {
        amount
      }
      argument {
        id
        creator {
          address
        }
        timestamp {
          createdTime
        }
        storyId
        body
        likes {
          argumentID
          creator {
            address
          }
          timestamp {
            createdTime
          }
        }
      }
    }
  }
}
```

