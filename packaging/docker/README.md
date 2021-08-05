# Docker compose

The docker-compose provides an app container for Pen itself and a signer container for Clef.
To prepare your machine to run docker compose execute
```
mkdir -p pen && cd pen
wget -q https://raw.githubusercontent.com/ethersphere/pen/master/packaging/docker/docker-compose.yml
wget -q https://raw.githubusercontent.com/ethersphere/pen/master/packaging/docker/env -O .env
```
Set all configuration variables inside `.env`

`clef` is configured with `CLEF_CHAINID=5` for goerli

Pen requires an Ethereum endpoint to function. Obtain a free Infura account and set:
- `PEN_SWAP_ENDPOINT=wss://goerli.infura.io/ws/v3/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

Set pen password by either setting `PEN_PASSWORD` or `PEN_PASSWORD_FILE`

If you want to use password file set it to
- `PEN_PASSWORD_FILE=/password`

Mount password file local file system by adding
```
- ./password:/password
```
to pen volumes inside `docker-compose.yml`

Start it with
```
docker-compose up -d
```

From logs find URL line with `on goerli you can get both goerli eth and goerli pen from` and prefund your node
```
docker-compose logs -f pen-1
```

Update services with
```
docker-compose pull && docker-compose up -d
```

## Running multiple Pen nodes
It is easy to run multiple pen nodes with docker compose by adding more services to `docker-compose.yaml`
To do so, open `docker-compose.yaml`, copy lines 3-58 and past this after line 58.
In the copied lines, replace all occurences of `pen-1` with `pen-2`, `clef-1` with `clef-2` and adjust the `API_ADDR` and `P2P_ADDR` and `DEBUG_API_ADDR` to respectively `1733`, `1734` and `127.0.0.1:1735`
Lastly, add your newly configured services under `volumes` (last lines), such that it looks like:
```yaml
volumes:
  clef-1:
  pen-1:
  pen-2:
  clef-2:
```

If you want to create more than two nodes, simply repeat the process above, ensuring that you keep unique name for your pen and clef services and update the ports