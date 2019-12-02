pkill -f Peerster
go build
cd _Downloads
rm *
cd ..
cd client/
go build
cd ..

./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -N=5 -hw3ex3 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -N=5 -hw3ex3 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5004 -name=C -UIPort=8003 -N=5 -hw3ex3 > C.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5004 -gui -GUIPort=8084 -peers=127.0.0.1:5003 -name=D -UIPort=8004 -N=5 -hw3ex3 > D.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5005 -gui -GUIPort=8085 -peers=127.0.0.1:5003 -name=E -UIPort=8005 -N=5 -hw3ex3 > E.txt &

sleep 2
cd client/
./client -UIPort=8001 -file=QishanWang.png &
sleep 1
./client -UIPort=8002 -file=2.png &
sleep 1
./client -UIPort=8003 -file=Shaokang.png &
sleep 1
./client -UIPort=8005 -file=1.png &
sleep 1
./client -UIPort=8001 -file=lake.jpg &
sleep 1
./client -UIPort=8003 -file=3.png &
sleep 1
./client -UIPort=8005 -file=4.png &
cd ..

sleep 20
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..