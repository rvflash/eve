FROM golang:1.10

RUN mkdir -p /go/src/github.com/rvflash/eve
ADD . /go/src/github.com/rvflash/eve

# Manages dependencies with do dep.
WORKDIR /go/src/github.com/rvflash/eve
RUN go get -u github.com/golang/dep/...
RUN dep ensure -vendor-only

# In-memory cache via RPC.
WORKDIR /go/src/github.com/rvflash/eve/server/tcp
RUN go build

# IHM to manage EVE via HTTP.
WORKDIR /go/src/github.com/rvflash/eve/server/http
RUN go build