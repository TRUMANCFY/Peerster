pkill -f Peerster
go build
cd _Downloads
rm *
cd ..
cd client/
go build
cd ..

# ./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -rtimer=1 > A.txt &
# sleep 1
# ./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -rtimer=1 > B.txt &
# sleep 1
# ./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5004 -name=C -UIPort=8003 -rtimer=1 > C.txt &
# sleep 1
# ./Peerster -gossipAddr=127.0.0.1:5004 -gui -GUIPort=8084 -peers=127.0.0.1:5005 -name=D -UIPort=8004 -rtimer=1 > D.txt &
# sleep 1
# ./Peerster -gossipAddr=127.0.0.1:5005 -gui -GUIPort=8085 -peers=127.0.0.1:5004 -name=E -UIPort=8005 -rtimer=1 > E.txt &

# sleep 20
# cd client/
# ./client -UIPort=8001 -file=QishanWang.png &
# ./client -UIPort=8005 -file=Shaokang.png &

# sleep 1
# ./client -UIPort=8005 -dest=A -file=EgetfromA.png -request=469403655c3a182a6b7856052a2428ebd24fede9e39b6cb428c21b8a0c222cc4 &
# ./client -UIPort=8001 -dest=E -file=AgetfromE.png -request=2571718c9d1d4bbe9807df21f0dd84209d36b418ea15ca350c258495cdbe474d &
# ./client -UIPort=8003 -dest=E -file=CgetfromE.png -request=2571718c9d1d4bbe9807df21f0dd84209d36b418ea15ca350c258495cdbe474d &


# ./client -UIPort=8001 -dest=E -msg=PrivateFromAToE &
# ./client -UIPort=8005 -dest=A -msg=PrivateFromEToA &

./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -N=5 -hw3ex2 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -N=5 -hw3ex2 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5004 -name=C -UIPort=8003 -N=5 -hw3ex2 > C.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5004 -gui -GUIPort=8084 -peers=127.0.0.1:5003 -name=D -UIPort=8004 -N=5 -hw3ex2 > D.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5005 -gui -GUIPort=8085 -peers=127.0.0.1:5003 -name=E -UIPort=8005 -N=5 -hw3ex2 > E.txt &

sleep 10
cd client/
./client -UIPort=8001 -file=QishanWang.png &
./client -UIPort=8003 -file=Shaokang.png &
./client -UIPort=8005 -file=1.png &


cd ..

sleep 10
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..