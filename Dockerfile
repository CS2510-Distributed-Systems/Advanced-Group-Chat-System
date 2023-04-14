FROM ubuntu:latest

ENV CGO_ENABLED=0
# Install basic packages. use based on need.
RUN apt update
RUN apt install -y openssl ca-certificates vim make gcc golang-go protobuf-compiler python3 netcat iputils-ping iproute2 git

# Set up certificates
ARG cert_location=/usr/local/share/ca-certificates
RUN mkdir -p ${cert_location}
# Get certificate from "github.com"
RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt
# Get certificate from "proxy.golang.org"
RUN openssl s_client -showcerts -connect google.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/google.golang.crt
# Update certificates
RUN update-ca-certificates

# Install go extensions for protoc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ENV PATH="$PATH:/go/bin"


#copying the project to docker

COPY Advanced-Group-Chat-System /app/chat-system
ENV APP_HOME /app/chat-system
WORKDIR "${APP_HOME}"

#clean
RUN make clean

#build the go project
RUN make project
WORKDIR "${APP_HOME}"

#install dependencies
RUN go mod download

EXPOSE 12000

#build the server
RUN make -B server 

#build the client
RUN make -B client



