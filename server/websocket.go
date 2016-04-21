package server

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	updatePeriod = 2 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func getTeamScoreIfModified(lastMod time.Time) ([]Result, time.Time, error) {
	mod := DataGetLastResult()
	if !mod.After(lastMod) {
		return nil, lastMod, nil
	}
	r := DataGetAllScore()
	return r, mod, nil
}

func getServiceIfModified(lastMod time.Time) ([]interface{}, time.Time, error) {
	mod := DataGetLastServiceResult()
	if !mod.After(lastMod) {
		return nil, lastMod, nil
	}
	r := DataGetServiceStatus()
	/*
		if len(r) == 0 {
			return nil, mod, nil
		}
	*/
	return r, mod, nil
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, lastMod time.Time, which string) {
	pingTicker := time.NewTicker(pingPeriod)
	updateTicker := time.NewTicker(updatePeriod)
	defer func() {
		pingTicker.Stop()
		updateTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-updateTicker.C:
			var r []Result
			var t []interface{}
			var err error

			if which == "score" {
				r, lastMod, err = getTeamScoreIfModified(lastMod)
			} else if which == "service" {
				t, lastMod, err = getServiceIfModified(lastMod)
			}
			if err != nil {
				Logger.Printf("Could not get websocket team score: %v\n", err)
			}

			if r != nil || t != nil {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if which == "service" {
					if err := ws.WriteJSON(t); err != nil {
						return
					}
				} else {
					if err := ws.WriteJSON(r); err != nil {
						return
					}
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func ServeServicesWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			Logger.Println(err)
		}
		return
	}

	go writer(ws, DataGetLastServiceResult(), "service")
	reader(ws)
}

func ServeScoresWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			Logger.Println(err)
		}
		return
	}

	go writer(ws, DataGetLastResult(), "score")
	reader(ws)
}
