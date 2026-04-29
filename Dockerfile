# get golang container
FROM golang:1.26.0 AS builder

# get args
ARG apiVersion=unknown

# create and set workingfolder
WORKDIR /go/src/

# 与仓库布局一致：go.mod 在根目录，可执行包在 src/（含子包 src/docs，import 为 .../teslamateapi/src/docs）
COPY go.mod go.sum ./
COPY src ./src

# download go mods and compile the program（./src 为 package main；勿用 ./src/... 否则会同时编译 docs 包）
RUN go mod download && \
  CGO_ENABLED=0 GOOS=linux go build \
  -a -installsuffix cgo -ldflags="-w -s \
  -X 'main.apiVersion=${apiVersion}' \
  " -o app ./src


# get alpine container
FROM alpine:3.23.3 AS app

# create workdir
WORKDIR /opt/app

# add packages, create nonroot user and group
RUN apk --no-cache add ca-certificates tzdata && \
  addgroup -S nonroot && \
  adduser -S nonroot -G nonroot && \
  chown -R nonroot:nonroot .

# set user to nonroot
USER nonroot:nonroot

# copy binary from builder
COPY --from=builder --chown=nonroot:nonroot --chmod=555 /go/src/app .

# expose port 8080
EXPOSE 8080

# run application
CMD ["./app"]
