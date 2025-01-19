// client.go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    pb "fern-reporter/ping"
    gtid "fern-reporter/gettestrunbyid"

)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := pb.NewPingServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    r, err := c.Ping(ctx, &pb.PingRequest{Message: "Ping"})
    if err != nil {
        log.Fatalf("could not ping: %v", err)
    }
    log.Printf("Ping response: %s", r.GetMessage())

    // implementation for gtid

    d := gtid.NewTestRunServiceClient(conn)

    ctx, cancel = context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    k, err := d.GetTestRunByID(ctx, &gtid.GetTestRunByIDRequest{Id: "1"})
    if err != nil {
        log.Fatalf("could not get test run: %v", err)
    }
    log.Printf("TestRun: %v", k.GetTestRun())
}

