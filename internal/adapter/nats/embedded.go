package nats

import (
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// RunEmbeddedServer starts an in-process NATS server and returns a client
// connected to it. Used by benchmarks and local experiments.
func RunEmbeddedServer(inProcess bool, enableLogging bool) (*nats.Conn, *server.Server, error) {
	opts := &server.Options{
		ServerName:      "embedded_server",
		DontListen:      inProcess,
		JetStream:       true,
		JetStreamDomain: "embedded",
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, err
	}

	if enableLogging {
		ns.ConfigureLogger()
	}
	go ns.Start()

	if !ns.ReadyForConnections(5 * time.Second) {
		return nil, nil, err
	}

	clientOpts := []nats.Option{}
	if inProcess {
		clientOpts = append(clientOpts, nats.InProcessServer(ns))
	}

	nc, err := nats.Connect(nats.DefaultURL, clientOpts...)
	if err != nil {
		return nil, nil, err
	}

	return nc, ns, err
}
