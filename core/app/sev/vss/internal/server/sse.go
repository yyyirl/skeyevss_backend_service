package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"skeyevss/core/app/sev/vss/internal/handler/sse"
	"skeyevss/core/app/sev/vss/internal/types"
	"skeyevss/core/pkg/dt"
	"skeyevss/core/pkg/functions"
)

type SSWSev struct {
	svcCtx *types.ServiceContext
}

func NewSSESev(svcCtx *types.ServiceContext) *SSWSev {
	return &SSWSev{
		svcCtx: svcCtx,
	}
}

func (l *SSWSev) Start() {
	var addr = fmt.Sprintf("%s:%d", l.svcCtx.Config.Host, l.svcCtx.Config.SSE.Port)
	functions.PrintStyle("blue", "SSE Events Listen on: ", addr)
	http.HandleFunc("/events", l.handler)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

func (l *SSWSev) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var buf = l.svcCtx.Config.SSE.MessageChanBuffer
	if buf <= 0 {
		buf = 256
	}
	var (
		messageChan = make(chan *types.SSEResponse, buf)
		afterClose  = false
	)
	defer func() {
		functions.LogInfo("event connection closed")
		if afterClose {
			dt.SetTimeout(
				2*time.Second,
				func() {
					close(messageChan)
				},
			)
			return
		}

		close(messageChan)
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sse.RegisterRouter(ctx, r, l.svcCtx, messageChan)

	for item := range messageChan {
		afterClose = item.DelayClose
		content, flush := l.toResp(item)
		if flush {
			if content != "" {
				_, _ = fmt.Fprintf(w, content)
				w.(http.Flusher).Flush()
			}
			continue
		}

		if content != "" {
			_, _ = fmt.Fprintf(w, content)
		}
		break
	}
}

func (l *SSWSev) toResp(res *types.SSEResponse) (string, bool) {
	if res.Err != nil {
		return fmt.Sprintf("event: end\ndata: {\"error\": \"%s\"}\n\n", res.Err.Error), false
	}

	if res.Data != nil {
		b, err := functions.JSONMarshal(res.Data)
		if err != nil {
			return fmt.Sprintf("event: end\ndata: {\"error\": \"%s\"}\n\n", err), false
		}

		return fmt.Sprintf("data: {\"data\": %s}\n\n", b), true
	}

	if res.Done {
		return "event: end\ndata: {}\n\n", false
	}

	return "", false
}
