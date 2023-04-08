package main

import (
	"bufio"
	"cs2510/rpc_conc/strRPCLock"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

func main() {
	// Process commandline arguments
	portArg := flag.Int("port", 12000, "the port to listen for messages on")
	addrArg := flag.String("address", "localhost", "address of server")

	flag.Parse()

	port := *portArg
	serverAddr := *addrArg

	// Connect to RPC server
	client, err := rpc.Dial("tcp", serverAddr+":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("dialing:", err)
	}

	// Read from keyboard
	fmt.Print("Enter your message: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = input[:len(input)-1] // strip trailing '\n'

	// Set up arguments for RPC call
	appStr := "***"
	args := strRPCLock.StrModArgs{input, appStr}
	var reply string
	fmt.Printf("about to call ModifyString with:\n")
	fmt.Printf("    origStr = '%s', appStr = '%s'\n\n", input, appStr)

	// Do RPC call
	err = client.Call("StrMod.ModifyString", &args, &reply)
	if err != nil {
		log.Fatal(err)
	}

	// Print reply
	fmt.Println("reply: ", reply)

	// Do another RPC call (find out how many requests the server has processed
	// so far
	var count int
	count = 0
	err = client.Call("StrMod.GetReqCount", &count, &count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("requests processed: ", count)

	// Do many simultaneous RPC calls
	wg := new(sync.WaitGroup)
	for i := 1; i <= 10000; i++ {
		wg.Add(1)
		go func(myargs strRPCLock.StrModArgs) {
			var myreply string
			defer wg.Done()
			myerr := client.Call("StrMod.ModifyString", &myargs, &myreply)
			if myerr != nil {
				log.Fatal(myerr)
			}
		}(args)
	}
	wg.Wait()

	err = client.Call("StrMod.GetReqCount", &count, &count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("requests processed: ", count)

	client.Close()
}
