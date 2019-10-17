package main

import (
	"awesomeProject/protobuf"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"sync"
	"time"
)

var (
	wait   *sync.WaitGroup
	client protobuf.BroadcastClient
)

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *protobuf.User) error {
	var streamerror error

	stream, err := client.CreateStream(context.Background(), &protobuf.Connect{User: user, Active: true})
	if err != nil {
		return fmt.Errorf("connect failed: %v", err)
	}

	wait.Add(1)
	go func(str protobuf.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}
			fmt.Printf("%v : %s\n", msg.Id, msg.Content)
		}
	}(stream)
	return streamerror
}
func main() {
	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("N", "Anon", "The name of the user")
	flag.Parse()
	id := sha256.Sum256([]byte(timestamp.String() + *name))

	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("couldn't connect to server %v", err)
	}

	client = protobuf.NewBroadcastClient(conn)
	user := &protobuf.User{Id: hex.EncodeToString(id[:]), Name: *name}

	connect(user)
	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &protobuf.Message{
				Id:        user.Id,
				Content:   scanner.Text(),
				Timestamp: timestamp.String(),
			}
			_, err := client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				fmt.Printf("Error Sending Message: %v", err)
				break
			}
		}
	}()
	go func() {
		wait.Wait()
		close(done)
	}()
	<-done
}
