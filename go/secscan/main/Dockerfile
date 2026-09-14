FROM saichler/builder:latest AS build

COPY main.go /home/src/github.com/saichler/build/main.go
RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go build -o secscan

FROM saichler/secscan-postgres:latest AS final

COPY --from=build /home/src/github.com/saichler/build/secscan /home/run/secscan
COPY start.sh /home/run/start.sh
RUN chmod +x /home/run/start.sh

ENTRYPOINT ["/home/run/start.sh"]
