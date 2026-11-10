package main

import (
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestLiveChatTokenStreamingAndSaving(t *testing.T){
 app:=testApp(t,&fixtureProvider{})
 cookie:=login(t,app,"SCHOOL_TEST_KEY_DO_NOT_PERSIST")
 calls:=0
 gateway:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls++
  if r.Header.Get("api-key")!="SCHOOL_TEST_KEY_DO_NOT_PERSIST" {t.Error("School api-key authentication header missing")}
  if r.Header.Get("Accept")=="application/json" {t.Error("Unexpected nonstreaming accept")}
  w.Header().Set("Content-Type","text/event-stream")
  fmt.Fprint(w,"data: {\"choices\":[{\"delta\":{\"content\":\"Quick \"}}]}\n\n")
  fmt.Fprint(w,"data: {\"choices\":[{\"delta\":{\"content\":\"grounded answer [S1].\"},\"finish_reason\":\"stop\"}]}\n\n")
  fmt.Fprint(w,"data: [DONE]\n\n")
 }))
 defer gateway.Close()
 app.aiURL=gateway.URL
 w,_:=request(t,app,"POST","/api/chat/stream",ChatRequest{Message:"Assess NVDA",Symbol:"NVDA",Persona:"assistant"},cookie)
 if w.Code!=200 {t.Fatalf("SSE request failed: %d %s",w.Code,w.Body.String())}
 if calls!=1 {t.Fatalf("Expected one model request, got %d",calls)}
 if !strings.Contains(w.Header().Get("Content-Type"),"text/event-stream"){t.Fatal("Wrong SSE content type")}
 raw:=w.Body.String()
 for _,want:=range []string{"event: status","event: delta","event: complete","Quick","grounded answer [S1]"}{
  if !strings.Contains(raw,want){t.Errorf("Missing streamed fragment %q: %s",want,raw)}
 }
 if strings.Contains(raw,"SCHOOL_TEST_KEY_DO_NOT_PERSIST"){t.Fatal("Key must not appear in stream")}
 history:=app.store.snapshot().Messages
 if len(history)!=2||history[1].Content!="Quick grounded answer [S1]."{t.Fatalf("Chat history did not save streamed answer: %+v",history)}
}
