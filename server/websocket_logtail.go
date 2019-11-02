package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func WsTailFile(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	ctx := c.CloseRead(r.Context())

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Logger.WithError(err).Error("WsTailFile.NewWatcher failed")
		wsjson.Write(ctx, c, map[string]interface{}{"error": "Internal Server Error"})
		return
	}
	defer watcher.Close()

	path := chi.URLParam(r, "name")
	path = filepath.Join(LogDir, filepath.Clean("/"+path))

	fd, err := os.Open(path)
	if err != nil {
		Logger.WithError(err).WithField("path", path).Error("WsTailFile: file open failed")
		wsjson.Write(ctx, c, map[string]interface{}{"error": fmt.Sprintf("%q: file does not exist", path)})
		return
	}
	defer fd.Close()

	fi, err := fd.Stat()
	if err != nil {
		Logger.WithError(err).WithField("path", path).Error("WsTailFile: file stat failed")
		wsjson.Write(ctx, c, map[string]interface{}{"error": "file read error"})
		return
	}
	sz := fi.Size()

	fd.Seek(sz, io.SeekCurrent)

	err = watcher.Add(path)
	if err != nil {
		Logger.WithError(err).WithField("path", path).Error("WsTailFile: watcher.Add failed")
		wsjson.Write(ctx, c, map[string]interface{}{"error": "Internal server error"})
		return
	}

	buf := bytes.NewBuffer(make([]byte, bytes.MinRead))
	for {
		buf.Reset()
		select {
		case <-ctx.Done():
			c.Close(websocket.StatusNormalClosure, "ctx done")
			return
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				fi, _ = fd.Stat()
				if sz == fi.Size() {
					continue
				}
				sz = fi.Size()

				_, err = buf.ReadFrom(fd)
				if err != nil {
					Logger.WithError(err).WithField("path", path).Error("file read failed")
					wsjson.Write(ctx, c, map[string]interface{}{"error": "Internal server error"})
					return
				}

				err = c.Write(ctx, websocket.MessageText, buf.Bytes())
				if err != nil {
					Logger.WithError(err).WithField("path", path).Error("write ws failed")
					return
				}

			}
		}
	}

}
