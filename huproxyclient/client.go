// Copyright 2018 David Marby
// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pkg/browser"

	huproxy "github.com/DMarby/huproxy/lib"
)

var (
	writeTimeout = flag.Duration("write_timeout", 10*time.Second, "Write timeout")
	verbose      = flag.Bool("verbose", false, "Verbose.")
)

func dialError(url string, resp *http.Response, err error) {
	if resp != nil {
		extra := ""
		if *verbose {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Failed to read HTTP body: %v", err)
			}
			extra = "Body:\n" + string(b)
		}
		log.Fatalf("%s: HTTP error: %d %s\n%s", err, resp.StatusCode, resp.Status, extra)

	}
	log.Fatalf("Dial to %q fail: %v", url, err)
}

func setupConnection(url string, token string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dialer := websocket.Dialer{}
	head := map[string][]string{}

	head["Authorization"] = []string{
		"Bearer " + token,
	}

	conn, resp, err := dialer.Dial(url, head)
	if err != nil {
		dialError(url, resp, err)
	}
	defer conn.Close()

	// websocket -> stdout
	go func() {
		for {
			mt, r, err := conn.NextReader()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if err != nil {
				log.Fatal(err)
			}
			if mt != websocket.BinaryMessage {
				log.Fatal("blah")
			}
			if _, err := io.Copy(os.Stdout, r); err != nil {
				log.Printf("Reading from websocket: %v", err)
				cancel()
			}
		}
	}()

	// stdin -> websocket
	// TODO: NextWriter() seems to be broken.
	if err := huproxy.File2WS(ctx, cancel, os.Stdin, conn); err == io.EOF {
		if err := conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(*writeTimeout)); err == websocket.ErrCloseSent {
		} else if err != nil {
			log.Printf("Error sending close message: %v", err)
		}
	} else if err != nil {
		log.Printf("reading from stdin: %v", err)
		cancel()
	}

	if ctx.Err() != nil {
		os.Exit(1)
	}
}

func retrieveAuthenticationToken() string {
	m := mux.NewRouter()

	s := &http.Server{
		Addr:    "127.0.0.1:8087",
		Handler: m,
	}

	channel := make(chan string)

	m.HandleFunc("/callback/{token}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		fmt.Fprint(w, "Done authenticating")
		channel <- token
	})

	go func() {
		s.ListenAndServe()
	}() // TODO: Can we simplify to not wrap in func?

	browser.OpenURL("https://ssh.home.dmarby.se/auth") // TODO: Don't hardcode url

	return <-channel
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("Want exactly one arg")
	}
	url := flag.Arg(0)

	if *verbose {
		log.Printf("huproxyclient %s", huproxy.Version)
	}

	log.Printf("huproxy %s", huproxy.Version)
	token := retrieveAuthenticationToken() // TODO: error handling

	setupConnection(url, token)
}
