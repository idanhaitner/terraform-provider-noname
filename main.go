package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/idanhaitner/terraform-provider-noname/internal/provider"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	serverFactory, err := provider.ProtoV5ProviderServerFactory(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt

	if *debugFlag {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	logFlags := log.Flags()
	logFlags = logFlags &^ (log.Ldate | log.Ltime)
	log.SetFlags(logFlags)

	err = tf5server.Serve(
		"hashicorp.com/edu/hashicups",
		serverFactory,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
