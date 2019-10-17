package main

import (
	"awesomeProject/protobuf"
	"context"
	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
	"log"
	"net"
	"os"
	"sync"
)

func init() {
	glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

type Connection struct {
	Stream protobuf.Broadcast_CreateStreamServer
	ID     string
	Active bool
	error  chan error
}

type Server struct {
	Connection []*Connection
}

func (s Server) CreateStream(pconn *protobuf.Connect, stream protobuf.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		Stream: stream,
		ID:     pconn.User.Id,
		Active: pconn.Active,
		error:  make(chan error),
	}
	s.Connection = append(s.Connection, conn)
	return <-conn.error
}

func (s Server) BroadcastMessage(ctx context.Context, message *protobuf.Message) (*protobuf.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connection {
		wait.Add(1)

		go func(msg *protobuf.Message, conn *Connection) {
			defer wait.Done()

			if conn.Active {
				err := conn.Stream.Send(msg)
				glog.Info("Send message to: ", conn.Stream)
				if err != nil {
					glog.Error("Error with steam: %s - Error: %v", conn.Stream, err)
					conn.Active = false
					conn.error <- err
				}
			}
		}(message, conn)
	}
	go func() {
		wait.Wait()
		close(done)
	}()
	<-done
	return &protobuf.Close{}, nil
}

func main() {
	var connections []*Connection

	server := &Server{connections}

	grpcServer := grpc.NewServer()
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("error creating the server %v", err)
	}
	glog.Info("Starting server at port :8080")

	protobuf.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(listen)
}
