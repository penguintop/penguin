FROM golang:1.15 AS build

WORKDIR /src
# enable modules caching in separate layer
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN make binary

FROM debian:10.9-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r pen --gid 999; \
    useradd -r -g pen --uid 999 --no-log-init -m pen;

# make sure mounted volumes have correct permissions
RUN mkdir -p /home/pen/.pen && chown 999:999 /home/pen/.pen

COPY --from=build /src/dist/pen /usr/local/bin/pen

EXPOSE 1633 1634 1635
USER pen
WORKDIR /home/pen
VOLUME /home/pen/.pen

ENTRYPOINT ["pen"]
