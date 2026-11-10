package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// emitStreamEvent serializes only structured JSON to prevent multiline
// injection into SSE frames and flushes each response fragment promptly.
func emitStreamEvent(w http.ResponseWriter, flush http.Flusher, name string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil { return err }
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, raw); err != nil {return err}
	flush.Flush()
	return nil
}

func (a *App) streamChat(w http.ResponseWriter, r *http.Request) {
	var input ChatRequest
	if err := decode(w,r,&input);err!=nil {fail(w,400,err);return}
	if a.getKey()=="" {
		fail(w,400,errors.New("Enter your ENT207 DeepSeek API key in Settings first"))
		return
	}
	flusher,ok:=w.(http.Flusher)
	if !ok {fail(w,500,errors.New("This server does not support streaming responses"));return}
	w.Header().Set("Content-Type","text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control","no-cache, no-store")
	w.Header().Set("X-Accel-Buffering","no")
	w.Header().Set("Connection","keep-alive")
	if err:=emitStreamEvent(w,flusher,"status",map[string]string{"text":"Collecting live evidence…"});err!=nil{return}
	ctx,cancel:=context.WithTimeout(r.Context(),80*time.Second)
	defer cancel()
	result,err:=a.chatWithDeltas(ctx,input,a.sharedKey == "",func(delta string) error {
		return emitStreamEvent(w,flusher,"delta",map[string]string{"text":delta})
	})
	if err!=nil {
		_ = emitStreamEvent(w,flusher,"error",map[string]string{"error":err.Error()})
		return
	}
	_ = emitStreamEvent(w,flusher,"complete",result)
}
