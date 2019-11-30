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
./Peerster -gossipAddr=127.0.0.1:5005 -peers=127.0.0.1:5004 -name=E -UIPort=8005 -rtimer=1 > E.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5006 -peers=127.0.0.1:5005 -name=F -UIPort=8006 -rtimer=1 > F.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5007 -peers=127.0.0.1:5005 -name=G -UIPort=8007 -rtimer=1 > G.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5008 -peers=127.0.0.1:5006 -name=H -UIPort=8008 -rtimer=1 > H.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5009 -peers=127.0.0.1:5008 -name=I -UIPort=8009 -rtimer=1 > I.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5010 -peers=127.0.0.1:5009 -name=J -UIPort=8010 -rtimer=1 > J.txt &

sleep 20
cd client/
./client -UIPort=8001 -file=QishanWang.png &
./client -UIPort=8008 -file=Shaokang.png &
./client -UIPort=8010 -file=Shaokang.png &

sleep 2
# ./client -UIPort=8003 -keywords=a,n -budget=8 &
./client -UIPort=8003 -keywords=ang,shsh &

sleep 5
./client -UIPort=8003 -file=test.png -request=2571718c9d1d4bbe9807df21f0dd84209d36b418ea15ca350c258495cdbe474d &
./client -UIPort=8003 -file=test2.png -request=469403655c3a182a6b7856052a2428ebd24fede9e39b6cb428c21b8a0c222cc4 &
cd ..

sleep 20
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..