package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/candango/httpok"
	"github.com/candango/httpok/middleware"
	"github.com/candango/httpok/session"
)

var template string = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>HttpOK Test Application</title>
</head>
<body style="background-color: #000000; color: #ffffff;">
<div class="container">

<h1>Session</h1>
<p>Counter %d</p>
<p>ID: %s</p>
<p>Origin: </p>
<b>Count ????</b>
<br>

</div>
<footer class="my-footer">HttpOK Test Application. Candango Open Source Group</footer>
</body>
</html>`

func main() {
	mux := http.NewServeMux()

	store := session.NewMemoryStore()
	// store := session.NewFileStore()
	s := session.NewStoreEngine(store, session.WithProperties(
		&session.EngineProperties{
			AgeLimit:      7 * time.Second,
			PurgeDuration: 15 * time.Second,
		},
	))
	s.Properties().Name = "FIRENADOSESSID"
	// s := session.NewFileEngine()
	err := s.Start(context.Background())
	if err != nil {
		panic(err)
	}

	// ctx, cancel := context.WithCancel(context.Background())
	reqCount := 0
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			reqCount++
		}()
		sess, err := session.SessionFromContext(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("error: %v", err)
			return
		}
		count := 0
		ok, err := sess.Has("count")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("error: %v", err)
			return
		}
		if ok {
			cs, err := sess.Get("count")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error: %v", err)
				return
			}
			count = int(cs.(float64))
		}
		log.Printf("%v", sess.Data)

		log.Printf("Request %d received from %s/n", reqCount, r.URL.Path)
		fmt.Fprintf(w, template, count, sess.Id)
		count++
		sess.Set("count", count)
	})

	srv := &http.Server{
		Addr: ":8887",
		Handler: middleware.Chain(
			middleware.ExactPath("/", mux),
			middleware.Logging(nil),
			middleware.Sessioned(s),
		),
	}

	gs := httpok.NewGracefulServer(
		srv,
		"session-test-server",
	)

	gs.Run()

	// fmt.Println("")
	// ticker := time.NewTicker(1 * time.Second)
	//
	// done := make(chan struct{})
	//
	// count := 0
	//
	// go func() {
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			fmt.Println("buu")
	// 			count++
	// 			if count > 5 {
	// 				done <- struct{}{}
	// 			}
	// 		}
	// 	}
	// }()
	//
	// <-done
	//
	// fmt.Println("Processamento terminado")

	// sess := security.RandomString(64)
	//
	// err = engine.Store(sess, map[string]any{
	// 	"key": "value",
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// var sessData map[string]any
	// err = engine.Read(sess, &sessData)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// defer func() {
	// 	sessFile := filepath.Join(engine.Dir, fmt.Sprintf("%s.sess", sess))
	// 	err := os.Remove(sessFile)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	//
	// }()
}
