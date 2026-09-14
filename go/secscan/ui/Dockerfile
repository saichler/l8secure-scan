FROM saichler/builder:latest AS build

COPY ./main/main.go /home/src/github.com/saichler/build/

RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go build -o secscanweb

FROM saichler/secscan-security:latest AS final
COPY --from=build /home/src/github.com/saichler/build/secscanweb /home/run/secscanweb
COPY ./web /home/run/web

ENTRYPOINT ["/home/run/secscanweb"]
