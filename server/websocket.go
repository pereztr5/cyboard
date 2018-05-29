package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Poll the db for updates, using broadcastHub.timeCheck() at the end of each period.
	updatePeriod = 5 * time.Second

	// Deadline for write operations to any client. If missed, the client is dropped.
	writeWait = 10 * time.Second

	// Send pings to clients with this period. If missed, the client is dropped.
	pingPeriod = 10 * time.Second
)

// broadcastHub allows a form of pub/sub messaging, in which many subscribers
// listen to data from a single publisher. Whenever new data is received, the
// hub will prepare the data once, and then send it to each client.
type broadcastHub struct {
	logID   string
	closeCh chan struct{}
	conns   map[*websocket.Conn]chan *websocket.PreparedMessage

	// Lock for the conns map. Could use sync.Map from Go v1.9, but supporting v1.8
	// is nice for distros such as CentOS and Ubuntu.
	*sync.RWMutex

	timeCheck  func() (time.Time, error)
	getPayload func() (interface{}, error)
}

func NewBroadcastHub(logID string, timeCheck func() (time.Time, error), getPayload func() (interface{}, error)) *broadcastHub {
	return &broadcastHub{
		logID:      logID,
		timeCheck:  timeCheck,
		getPayload: getPayload,

		closeCh: make(chan struct{}),
		conns:   make(map[*websocket.Conn]chan *websocket.PreparedMessage),
		RWMutex: &sync.RWMutex{},
	}
}

func (b *broadcastHub) logError(args ...interface{}) {
	Logger.WithField("bcast", b.logID).Errorln(args...)
}

func (b *broadcastHub) addClient(ws *websocket.Conn) chan *websocket.PreparedMessage {
	msgCh := make(chan *websocket.PreparedMessage, 1)
	b.Lock()
	b.conns[ws] = msgCh
	b.Unlock()
	return msgCh
}

func (b *broadcastHub) delClient(ws *websocket.Conn) {
	ws.Close()
	b.Lock()
	delete(b.conns, ws)
	b.Unlock()
}

// Stop sends a signal to the rest of the broadcastHub to shutdown.
// It will then clean up on its own, after a short period.
func (b *broadcastHub) Stop() {
	close(b.closeCh)
}

// Start kicks off the broadcastHub's polling service, that connects
// to the backend and waits for new data. When data shows up, the
// broadcastHub will send it to each client. New clients are added in
// the ServeWs method.
func (b *broadcastHub) Start() {
	updateTicker := time.NewTicker(updatePeriod)
	defer updateTicker.Stop()

	ts, _ := b.timeCheck()
	for {
		select {
		case <-updateTicker.C:
			newTs, err := b.timeCheck()
			if err != nil {
				b.logError("failed timeCheck: ", err)
				continue
			}
			if ts.Equal(newTs) {
				// no update
				continue
			}

			ts = newTs
			payload, err := b.getPayload()
			if err != nil {
				b.logError("failed getPayload:", err)
				continue
			}

			pm, err := prepPayload(payload)
			if err != nil {
				b.logError("failed prepPayload:", err)
				continue
			}

			b.RLock()
			for _, msgCh := range b.conns {
				select {
				case msgCh <- pm:
				default: // Skip clients that can't take the message immediately.
				}
			}
			b.RUnlock()
		case <-b.closeCh:
			// close all connections
			for ws := range b.conns {
				b.delClient(ws)
			}
		}
	}
}

func prepPayload(payload interface{}) (*websocket.PreparedMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return websocket.NewPreparedMessage(websocket.TextMessage, data)
}

// ServeWs returns a http.HandlerFunc-ready function, which processes new
// connections on a WebSocket client, and adds them as subscribers to the
// broadcastHub.
func (b *broadcastHub) ServeWs() http.Handler {
	// TODO: check that the write buffer size is a good fit (it's not)
	upgrader := websocket.Upgrader{
		ReadBufferSize:    32,   // We don't care about reads
		WriteBufferSize:   2560, // 2.5KB
		EnableCompression: true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				b.logError("handshake failed", err)
			}
			return
		}

		// Handle the new client
		msgCh := b.addClient(ws)
		go func(b *broadcastHub, ws *websocket.Conn, msgCh chan *websocket.PreparedMessage) {
			defer b.delClient(ws)
			pingTicker := time.NewTicker(pingPeriod)
			defer pingTicker.Stop()

			for {
				select {
				case msg, ok := <-msgCh:
					if !ok {
						return
					}
					ws.SetWriteDeadline(time.Now().Add(writeWait))
					if err := ws.WritePreparedMessage(msg); err != nil {
						b.logError(ws.RemoteAddr(), "write failed:", err)
						return
					}
				case <-pingTicker.C:
					ws.SetWriteDeadline(time.Now().Add(writeWait))
					if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
						return
					}
				case <-b.closeCh:
					return
				}
			}
		}(b, ws, msgCh)

		// Always discard messages sent from a client. On err (such as close), it will clean up.
		go func(b *broadcastHub, ws *websocket.Conn) {
			defer b.delClient(ws)
			for {
				if _, _, err := ws.NextReader(); err != nil {
					ws.Close()
					break
				}
			}
		}(b, ws)
	})
}

// ServiceStatusWsServer is a hub suitable for updating the Service Monitor.
func ServiceStatusWsServer() *broadcastHub {
	b := NewBroadcastHub(
		"service status",
		func() (time.Time, error) { return DataGetLastServiceResult(), nil },
		func() (interface{}, error) { return DataGetServiceStatus(), nil },
	)
	go b.Start()
	return b
}

// TeamScoreWsServer is a hub suitable for updating the Scoreboard charts & tables.
func TeamScoreWsServer() *broadcastHub {
	b := NewBroadcastHub(
		"team scores",
		func() (time.Time, error) { return DataGetLastResult(), nil },
		func() (interface{}, error) { return DataGetAllScoreSplitByType(), nil },
	)
	go b.Start()
	return b
}
