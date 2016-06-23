package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tchap/steemwatch/config"
	"github.com/tchap/steemwatch/notifications"
	"github.com/tchap/steemwatch/server"

	"github.com/go-steem/rpc"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

func main() {
	if err := _main(); err != nil {
		log.Fatalf("Error: %+v", err)
	}
}

func _main() error {
	// Load config from the environment.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Connect to MongoDB.
	mongo, err := mgo.Dial(cfg.MongoURL)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to MongoDB using %v", cfg.MongoURL)
	}
	defer mongo.Close()
	db := mongo.DB("steemwatch")

	// Connect to steemd.
	client, err := rpc.Dial(cfg.SteemdRPCEndpointAddress)
	if err != nil {
		return errors.Wrapf(
			err, "failed to connect to steemd using %v", cfg.SteemdRPCEndpointAddress)
	}
	defer client.Close()

	// Start catching signals.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// XXX: Not the greatest ideas to share MongoDB session.
	//      In case it is closed from one component, the other panics.

	// Start the block processor.
	notificationsCtx, err := notifications.Run(client, db)
	if err != nil {
		return err
	}

	// Start the web server.
	serverCtx, err := server.Run(db, cfg)
	if err != nil {
		return err
	}

	// Start processing signals.
	go func() {
		<-signalCh
		signal.Stop(signalCh)
		log.Println("Signal received, exiting...")
		notificationsCtx.Interrupt()
		serverCtx.Interrupt()
	}()

	var crashed bool

	if err := notificationsCtx.Wait(); err != nil {
		log.Printf("Notifications error: %+v", err)
		crashed = true
	}

	if err := serverCtx.Wait(); err != nil {
		log.Printf("Web server error: %+v", err)
		crashed = true
	}

	if crashed {
		return errors.New("crashed")
	}
	return nil
}
