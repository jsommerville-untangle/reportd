package messenger

import (
	"context"
	"unsafe"

	zmq "github.com/pebbe/zmq4"
	"github.com/untangle/golang-shared/services/logger"
	"github.com/untangle/reportd/services/monitor"
)

var messengerRelation = monitor.RoutineContextGroup{}

// Startup intializes the ZMQ socket and starts the sessionListener go routine
func Startup() {
	logger.Info("Setting up zmq listening socket...\n")
	socket, err := setupZmqSocket()
	if err != nil {
		logger.Warn("Unable to setup ZMQ sockets.")
	}

	messengerRelation = monitor.CreateRoutineContextRelation(context.Background(), "messenger", []string{"session_listener", "session_stats_listener", "interface_stats_listener"})

	logger.Info("Setting up event listeners on zmq socket...\n")
	go messageListener(messengerRelation.Contexts["session_listener"], "session_listener", "untangle:packetd:sessions", socket)
	go messageListener(messengerRelation.Contexts["session_stats_listener"], "session_stats_listener", "untangle:packetd:session-stats", socket)
	go messageListener(messengerRelation.Contexts["interface_stats_listener"], "interface_stats_listener", "untangle:packetd:interface-stats", socket)

}

// Shutdown signals the serviceShutdown channel to close any running goroutines spawned by this service
func Shutdown() {
	logger.Info("Shutting down messenger service...\n")
	monitor.CancelContexts(messengerRelation)
}

// messageListener is used to listen for ZMQ events being published
// THIS IS A ROUTINE
func messageListener(ctx context.Context, rtName string, topic string, soc *zmq.Socket) {
	monitor.RoutineStarted(rtName)
	defer monitor.RoutineEnd(rtName)
	defer soc.Close()

	for {

		select {
		case <-ctx.Done():
			logger.Info("Stopping ZMQ listener\n")
			return
		default:
			err := soc.SetSubscribe(topic)
			if err != nil {
				logger.Warn("Unable to subscribe to topic. \n")
				monitor.RoutineError(rtName)
				return
			}

			msg, err := soc.RecvMessageBytes(0)

			if err != nil {
				logger.Warn("Unable to receive messages: %s\n", err)
				monitor.RoutineError(rtName)
				return
			}

			logger.Info("Incoming Message on topic: %s size: %d bytes\n", topic, len(msg[1])+int(unsafe.Sizeof(msg[1])))

			// TODO:
			// Send this to an event router queue for parsing
			// Try to parse the message, if we cant then continue to next message
			//newEvt := &pbe.SessionEvent{}
			//if err := proto.Unmarshal(msg[1], newEvt); err != nil {
			//	logger.Warn("Unable to parse message: %s\n", err)
			//	continue
			//}
			//logger.Info("Converted message: %s\n", newEvt)

			//For now we will just unmarshal here and send to the report processing channels
			// TODO: really we should send this to some event router to determine if we need to populate both local and cloud channels
			//localreporting.AddToSessionChannel(newEvt)
			//outFunction(newEvt)
		}
	}
}

// setupZmqSocket prepares a zmq socket for listening to events
func setupZmqSocket() (soc *zmq.Socket, err error) {
	subscriber, err := zmq.NewSocket(zmq.SUB)

	if err != nil {
		logger.Err("Unable to open ZMQ socket... %s\n", err)
		return nil, err
	}

	// TODO: we should read a file created by packetd that contains a randomized
	// ZMQ port to listen on
	subscriber.Connect("tcp://localhost:5561")

	return subscriber, nil
}
