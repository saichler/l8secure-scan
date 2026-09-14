set -e

# clean up
rm -rf go.sum
rm -rf go.mod
rm -rf vendor

# fetch dependencies
go mod init
GOPROXY=direct GOPRIVATE=github.com go mod tidy
go mod vendor

# infrastructure: local postgres container matching secscan.json's
# credentials.postgres.creds.secscan {aside:secscan, yside:5432, zside:secscan}
docker rm -f unsecure-postgres 2>/dev/null || true
docker run -d --name unsecure-postgres -p 5432:5432 -v /data/:/data/ saichler/unsecure-postgres:latest secscan secscan secscan 5432

rm -rf demo
mkdir -p demo
cd secscan/log-agent
echo "Building log agent"
go build -o ../../demo/log-agent_demo
cd ../log-vnet
echo "Building log vnet"
go build -o ../../demo/log-vnet_demo
cd ../vnet
echo "Building vnet"
go build -o ../../demo/vnet_demo
cd ../main
echo "Building secscan"
go build -o ../../demo/secscan_demo
cd ../scanner
echo "Building scanner"
go build -o ../../demo/scanner_demo
cd ../ui/main
echo "Building ui"
go build -o ../../../demo/ui_demo
cd ..
cp -r ./web ../../demo/.
cd ../../tests/cmd
echo "Building mocks"
go build -o ../../demo/mocks_demo
cd ../../demo

echo "cd .." > ./kill_demo.sh
echo "rm -rf demo" >> ./kill_demo.sh
echo "docker rm -f unsecure-postgres" >> ./kill_demo.sh
echo "pkill -9 demo" >> ./kill_demo.sh
chmod +x ./kill_demo.sh

./log-vnet_demo &
./vnet_demo &
sleep 1
./log-agent_demo &
./secscan_demo &
sleep 2
./scanner_demo &
./ui_demo &
sleep 8
EXTERNAL_IP=$(ip route get 1.1.1.1 | grep -oP 'src \K[0-9.]+')
read -p "Press Enter to upload mocks"
./mocks_demo --address https://${EXTERNAL_IP}:2790 --user opsadmin --password opsadmin --insecure

read -p "Press Enter to kill the demo"
./kill_demo.sh
