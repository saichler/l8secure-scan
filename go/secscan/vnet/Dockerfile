FROM saichler/builder:latest AS build

ENV CGO_ENABLED 1
COPY main.go /home/src/github.com/saichler/build/main.go
RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go mod vendor

WORKDIR /
RUN /patch.sh

WORKDIR /home/src/github.com/saichler/build
RUN go build -o secscanvnet

FROM saichler/secscan-security:latest AS final
COPY --from=build /home/src/github.com/saichler/build/secscanvnet /home/run/secscanvnet
RUN mkdir -p /data
RUN chmod 777 /data
ENTRYPOINT ["/home/run/secscanvnet"]
