package server

import (
	"log"
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
	filePeriod = 10 * time.Second
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
	/*
		r, err := json.Marshal(DataGetAllScore())
		if err != nil {
			Logger.Printf("Error marshalling results: %v\n", err)
		}
	*/
	r := DataGetAllScore()
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

func writer(ws *websocket.Conn, lastMod time.Time) {
	pingTicker := time.NewTicker(pingPeriod)
	fileTicker := time.NewTicker(filePeriod)
	defer func() {
		pingTicker.Stop()
		fileTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-fileTicker.C:
			var r []Result
			var err error

			r, lastMod, err = getTeamScoreIfModified(lastMod)
			if err != nil {
				Logger.Printf("Could not get websocket team score: %v\n", err)
			}

			if r != nil {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteJSON(r); err != nil {
					return
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

func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go writer(ws, DataGetLastResult())
	reader(ws)
}
