package main

import (
	"fmt"
	"log"
	"time"

	r "gopkg.in/dancannon/gorethink.v2"
)

type hub struct {
	queue    queue
	channels channels
	ticker   *mTicker
	session     *r.Session
}

type channels map[string]*channel

func newHub() *hub {
	return &hub{
		queue:    make(queue, 16),
		channels: make(channels),
		ticker:   newMTicker(pingPeriod),
	}
}

func newChannel(h *hub, path string) *channel {
	return &channel{
		queue:       make(queue, 16),
		connections: make(connections),
		h:           h,
		path:        path,
	}
}

func (h *hub) run() {
	defer h.ticker.stop()

	// Open a connection to rethinkdb
	var err error
	h.session, err = r.Connect(r.ConnectOpts{
		Address: "localhost:28015",
		Database: "pinghub",
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer h.session.Close()

	for cmd := range h.queue {
		// Forward cmds to their path's channel queues.
		switch cmd.cmd {
		case SUBSCRIBE:
			h.subscribe(cmd)
		case PUBLISH:
			h.publish(cmd)
		case REMOVE:
			h.remove(cmd)
		default:
			panic(fmt.Sprintf("unexpected hub cmd: %v\n", cmd))
		}
	}
}

func (h *hub) subscribe(cmd command) {
	// Create a channel if needed.
	if _, ok := h.channels[cmd.path]; !ok {
		h.channels[cmd.path] = newChannel(h, cmd.path)
		go h.channels[cmd.path].run()
	}
	// Give the connection a reference to its own channel.
	cmd.conn.control <- h.channels[cmd.path]
	h.channels[cmd.path].queue <- cmd
}

func (h *hub) publish(cmd command) {
	if channel, ok := h.channels[cmd.path]; ok {
		select {
		case channel.queue <- cmd:
		default:
			// Tried publishing to a closing channel.
			h.remove(cmd)
		}
	} else {
		_, err := r.Table("pinghub").Insert(map[string]interface{}{
			"id": string(cmd.path),
			"text": string(cmd.text),
			"time": time.Now().UnixNano(),
		}, r.InsertOpts{
			Conflict: "replace",
		}).RunWrite(h.session)
		if err != nil {
			log.Println("Failed to publish post")
		}
	}
}

func (h *hub) remove(cmd command) {
	if _, ok := h.channels[cmd.path]; ok {
		delete(h.channels, cmd.path)
	}
}
