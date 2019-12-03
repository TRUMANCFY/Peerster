pkill -f Peerster
go build
cd _Downloads
rm *
cd ..
cd client/
go build
cd ..

./Peerster -gossipAddr=127.0.0.1:5001 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -rtimer=1 -hw3ex2 -gui -GUIPort=8081 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -rtimer=1 -hw3ex2 -gui -GUIPort=8082 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -peers=127.0.0.1:5004 -name=C -UIPort=8003 -rtimer=1 -hw3ex2 -gui -GUIPort=8083 > C.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5004 -peers=127.0.0.1:5003 -name=D -UIPort=8004 -rtimer=1 -hw3ex2 -gui -GUIPort=8084 > D.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5005 -peers=127.0.0.1:5004 -name=E -UIPort=8005 -rtimer=1 -hw3ex2 -gui -GUIPort=8085 > E.txt &
sleep 1

cd client
./client -UIPort=8003 -file=QishanWang.png &
./client -UIPort=8005 -file=Shaokang.png &
cd ..



sleep 1000
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..