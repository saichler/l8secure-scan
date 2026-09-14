FROM saichler/builder:latest AS build

COPY main.go /home/src/github.com/saichler/build/main.go
RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go build -o secscanscanner

FROM aquasec/trivy:latest AS trivy

FROM saichler/secscan-security:latest AS final
COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy
COPY --from=build /home/src/github.com/saichler/build/secscanscanner /home/run/secscanscanner
RUN mkdir -p /data
RUN chmod 777 /data

ENTRYPOINT ["/home/run/secscanscanner"]
