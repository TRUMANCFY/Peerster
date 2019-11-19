pkill -f Peerster
go build
cd _Downloads
rm *
cd ..
cd client/
go build
cd ..

./Peerster -gossipAddr=127.0.0.1:5001 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -rtimer=1 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -rtimer=1 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -peers=127.0.0.1:5004 -name=C -UIPort=8003 -rtimer=1 > C.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5004 -peers=127.0.0.1:5003 -name=D -UIPort=8004 -rtimer=1 > D.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5005 -peers=127.0.0.1:5003 -name=E -UIPort=8005 -rtimer=1 > E.txt &

sleep 20
cd client/
./client -UIPort=8001 -file=QishanWang.png &
./client -UIPort=8001 -file=Shaokang.png &

sleep 2
./client -UIPort=8003 -keywords=a,n -budget=8 &
cd ..

sleep 20
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..