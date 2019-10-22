#!/usr/bin/env bash


rm *.txt
go build
cd client
go build
cd ..


./Peerster -UIPort 8080 -gossipAddr 127.0.0.1:5000 -name A -peers 127.0.0.1:5001 > A.txt &
sleep 1
./Peerster -UIPort 8081 -gossipAddr 127.0.0.1:5001 -name B -peers 127.0.0.1:5002 > B.txt &
sleep 1
./Peerster -UIPort 8082 -gossipAddr 127.0.0.1:5002 -name C -peers 127.0.0.1:5000 > C.txt &
sleep 1

./client/client -msg juicy_rumor -UIPort 8080



sleep 20
pkill -f Peerster
