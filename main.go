package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tchap/steemwatch/config"
	"github.com/tchap/steemwatch/notifications"
	"github.com/tchap/steemwatch/notifications/notifiers/discord"
	"github.com/tchap/steemwatch/server"

	"github.com/go-steem/rpc"
	"github.com/go-steem/rpc/transports/websocket"
	"github.com/pkg/errors"
	"github.com/steemwatch/blockfetcher"
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
	wMongo, err := mgo.Dial(cfg.MongoURL)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to MongoDB using %v", cfg.MongoURL)
	}
	defer wMongo.Close()
	wDB := wMongo.DB("")

	nMongo := wMongo.Copy()
	defer nMongo.Close()
	nDB := nMongo.DB("")

	// Start catching signals.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the web server.
	serverCtx, dg, err := server.Run(wDB, cfg)
	if err != nil {
		return err
	}

	// Start notifications.
	notificationsCtx, client, err := runNotifications(nDB, cfg,
		notifications.SetWorkerCount(cfg.BlockProcessorWorkerCount),
		notifications.AddStandardNotifier("discord", discord.NewNotifier(dg)),
		notifications.AddNotifier("websocket", serverCtx.EventStreamManager))
	if err != nil {
		return err
	}
	defer client.Close()

	// Start processing signals.
	go func() {
		<-signalCh
		signal.Stop(signalCh)
		log.Println("Signal received, exiting...")

		serverCtx.Interrupt()

		if notificationsCtx != nil {
			notificationsCtx.Interrupt()
		}
	}()

	errCh := make(chan error, 2)

	go func() {
		var err error
		if notificationsCtx != nil {
			err = notificationsCtx.Wait()
			if err != nil {
				log.Printf("Notifications error: %+v", err)
			}
		}
		errCh <- err
	}()

	go func() {
		err := serverCtx.Wait()
		if err != nil {
			log.Printf("Web server error: %+v", err)
		}
		errCh <- err
	}()

	for i := 0; i < cap(errCh); i++ {
		if err := <-errCh; err != nil {
			return errors.New("crashed")
		}
	}
	return nil
}

func runNotifications(
	db *mgo.Database,
	cfg *config.Config,
	opts ...notifications.Option,
) (*blockfetcher.Context, *rpc.Client, error) {

	if cfg.SteemdDisabled {
		return nil, nil, nil
	}

	connect := func() (*rpc.Client, error) {
		// Monitor the connection to steemd.
		monitorChan := make(chan interface{})
		go func() {
			for event := range monitorChan {
				log.Println("steemd connection:", event)
			}
		}()

		// Connect to steemd.
		t, err := websocket.NewTransport(cfg.SteemdRPCEndpointAddress,
			websocket.SetAutoReconnectEnabled(true),
			websocket.SetAutoReconnectMaxDelay(1*time.Minute),
			websocket.SetMonitor(monitorChan))
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to connect to steemd using %v", cfg.SteemdRPCEndpointAddress)
		}
		client, err := rpc.NewClient(t)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate the steemd RPC client")
		}
		return client, nil
	}

	// Start the block processor.
	client, err := connect()
	if err != nil {
		return nil, nil, err
	}
	ctx, err := notifications.Run(client, connect, db, opts...)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	return ctx, client, nil
}
