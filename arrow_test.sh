pkill -f Peerster
go build

./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=B -UIPort=8002 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5002 -name=C -UIPort=8003 > C.txt &
sleep 1

sleep 100
pkill -f Peerster