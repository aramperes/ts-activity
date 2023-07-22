# ts-activity

This program will post notifications to Discord when someone joins or leaves your TeamSpeak server.

![image](https://github.com/aramperes/ts-activity/assets/6775216/bab942c3-b7d7-4b5e-8d14-4d69383bc856)


## Configuration

You will have to create ServerQuery credentials on an account that has permissions to login & view clients in the server. You can do this from the `Tools -> ServerQuery Login` menu in TeamSpeak 3.

This program is configured using environment variables:

- `TS_QUERY_ADDR`: Address to the TeamSpeak ServerQuery port. Example: `127.0.0.1:10011`
- `TS_QUERY_USER`: The username you selected for ServerQuery in the setup
- `TS_QUERY_PASS`: The password TeamSpeak generated for ServerQuery in the setup
- `TS_DISCORD_WEBHOOK`: Webhook URL for Discord. You can create this from the channel "Integrations" page

## Run it
[![docker hub](https://img.shields.io/docker/v/aramperes/ts-activity?color=%232496ed&label=docker%20hub&logo=docker&logoColor=fff&sort=semver)](https://hub.docker.com/r/aramperes/ts-activity)

To build and run locally:

```sh
go mod download
go run .
```

Or, using the [Docker image](https://hub.docker.com/r/aramperes/ts-activity):

```sh
docker run --rm --name ts-activity \
  -e TS_DISCORD_WEBHOOK='https://discord.com/api/webhooks/...' \
  -e TS_QUERY_ADDR=127.0.0.1:10011 \
  -e TS_QUERY_USER=Jeff \
  -e TS_QUERY_PASS=******* \
  aramperes/ts-activity
```

There is also a Helm chart. You can create a `Secret` containing `username`, `password`, and `discord`, and then:

```sh
helm repo add momoperes https://charts.momoperes.ca
helm repo update

helm upgrade --install ts-activity momoperes/ts-activity \
  --set config.serverQueryAddr=teamspeak:10011 \
  --set config.discordUsername=Jeff \
  --set config.serverQuerySecret=ts-activity \
  --set config.webhookSecret=ts-activity
```
