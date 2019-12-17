pkill -f Peerster
go build
cd _Downloads
rm *
cd ..
cd client/
go build
cd ..

./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001 -N=3 -hw3ex3 -rtimer=1 > A.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=B -UIPort=8002 -N=3 -hw3ex3 -rtimer=1 > B.txt &
sleep 1
./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5002 -name=C -UIPort=8003 -N=3 -hw3ex3 -rtimer=1 > C.txt &
sleep 1

cd client/
# ./client -UIPort=8002 -file=file1.png &
# sleep 1
# ./client -UIPort=8003 -file=file2.png &
# sleep 1
# ./client -UIPort=8002 -file=file3.png &
# sleep 1
# ./client -UIPort=8003 -file=1.png &
sleep 1
cd ..

sleep 100
pkill -f Peerster
rm Peerster
cd client
rm client
cd ..