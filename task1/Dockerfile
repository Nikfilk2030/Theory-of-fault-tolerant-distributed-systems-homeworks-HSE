FROM ubuntu:latest

RUN apt-get update && apt-get install -y build-essential cmake iptables

WORKDIR /app
COPY . .

RUN cmake . && make
