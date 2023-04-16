# Advanced-Group-Chat-System

A highly consistent, durable Group chat system

The Application is a replicated version of the group chat system with 5 servers

the system model assumed

1. Servers can crash
2. network may drop or arbitrarily delay

Steps to run the Application.

## Docker setup

Run the dockerfile which contains dependecies of the application

```
docker build . -t "cs2510_p2"
```

To connect the servers with one another, we need a bridge

```
docker network create --driver bridge cs2510
```

## Run the Server

we can start 5 servers with 5 different names of the docker containers over the network bridge.

```
docker run -it --network cs2510 --name server cs2510_p2
```

There are Two ways to start the server

1. To run the server manually
   Bash into the server container

```
docker exec -it server bash
```

Make file is written to run the server along with id

```
make server<id>
```

2. To use the script Test_p2.py

```
python3 test_p2.py init
```

## Run the client

Create a docker container for client over the network bridge

```
docker run -it --network cs2510 --name client cs2510_p2
```

Interacative Bash shell into the client container

```
docker exec -it client bash
```

run the client

```
./client_main
```

## client server Interaction

Once the client is running.
To connect to server and start app :

```
c <ip_address>:12000
```

# commands of the application

Once connected to server user will have to login by giving the following command.
-u <user_name>

j <group_name> :
 To join a group. If the given group doesn't exist, then a new group will be created and the user will be added to participants list.
If the user is already present in the group, then that user will be removed from the current group and will be added to the new group.

Now user can perform following actions after joined a group:
a <message\_> :
  This command is used to append message to the chat.
l <message_number>:
 This command is used to like a message. Here message id is the number displayed before every message. A user can not like his own message.
r <message_number>:
 This command is used to dislike a message. A user can only unlike a message iff the user had liked the same message beforehand.
p
  This command prints all the messages right from the group creation with latest message on bottom and oldest message on top.
q
 This command can be used at anytime after runs the client. This command terminates the client session and also removes
all information related from the server storage.
v
Provides the server's view at that moment i.e lists number of servers connected to the server connected to the client

Once after joining a group the user is shown with recent 10 chat messages of the group. User can use p command to see full chat history.
For every action performed by the user, all other active participants are broadcasted with the change.

# Development Usage of the Make File:

These are the commands that can be used to run the application:
-make clean
  This command clean all the files in pb folder which contains proto and grpc proto files.
-make gen
  This command generates proto and grpc proto files for the latest code.
-make server
  This command builds the server.
-make client
  This command builds client connection.
