package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lru "github.com/hashicorp/golang-lru"
)

var (
	listen   = flag.String("listen", ":8080", "HTTP Address to listen on")
	ncache   = flag.Int("cache", 256, "Number of cached items")
	external = flag.Bool("external", false, "Serve external index.html instead of bundling resources")

	cache      *lru.Cache
	resPath    string
	staticData []byte
)

func main() {
	flag.Parse()
	if *external {
		log.Println("Serve index file from GOPATH")

		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			log.Fatal("GOPATH is empty")
		}
		resPath = filepath.Join(gopath, "src/github.com/ng-vu/mermaid-service/index.html")

	} else {
		log.Println("Serve bundled resources")

		var err error
		staticData, err = indexHtmlBytes()
		if err != nil {
			panic(err)
		}
	}

	var err error
	cache, err = lru.New(*ncache)
	if err != nil {
		log.Fatalln("Unable to init cache", err)
	}

	http.HandleFunc("/", indexHandler)
	http.Handle("/diagram/", http.StripPrefix("/diagram/", http.HandlerFunc(diagramHandler)))

	log.Println("Server is listening on", *listen)
	err = http.ListenAndServe(*listen, nil)
	log.Fatalln("ListenAndServe", err)
}

const sampleURL = "/diagram/" +
	`Z3JhcGggVEQKQVtDaHJpc3RtYXNdIC0tPnxHZXQgbW9uZXl8IEIoR28gc2hvcHBpbmcpCkIgLS0-IEN7TGV0IG1lIHRoaW5rfQpDIC0tPnxPbmV8IERbTGFwdG9wXQpDIC0tPnxUd298IEVbaVBob25lXQpDIC0tPnxUaHJlZXwgRltDYXJdCg`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if resPath == "" {
		w.Header().Add("Content-Type", "text/html")
		_, _ = w.Write(staticData)
		return
	}

	http.ServeFile(w, r, resPath)
}

func getFromCache(s string) []byte {
	data, ok := cache.Get(s)
	if !ok {
		return nil
	}
	return data.([]byte)
}

func addToCache(s string, data []byte) {
	_ = cache.Add(s, data)
}

func diagramHandler(w http.ResponseWriter, req *http.Request) {
	log.Println("Handle", req.URL.Path)

	rawInput := req.URL.Path
	if rawInput == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
		return
	}

	if output := getFromCache(rawInput); output != nil {
		writeResponse(w, output)
		return
	}

	data, err := base64.RawURLEncoding.DecodeString(rawInput)
	if err != nil {
		log.Print("Unable to decode base64", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")
		return
	}

	output, err := generate(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Unable to handle request: %v", err)
		return
	}

	addToCache(rawInput, output)
	writeResponse(w, output)
}

func writeResponse(w http.ResponseWriter, output []byte) {
	w.Header().Add("Content-Type", "image/svg+xml")
	w.Header().Add("Cache-Control", "max-stale=31536000")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output)
}

func generate(input []byte) ([]byte, error) {
	f, err := ioutil.TempFile(os.TempDir(), "mm_")
	if err != nil {
		log.Println("Error creating temp file", err)
		return nil, err
	}
	defer func() {
		e := os.Remove(f.Name())
		if e != nil {
			log.Println("Unable to remove temp file", e)
		}
	}()

	_, err = f.Write(input)
	if err != nil {
		log.Println("Error writing file", err)
		return nil, err
	}
	defer func() {
		e := f.Close()
		if e != nil {
			log.Println("Unable to close temp file", e)
		}
	}()

	inputFile := f.Name()
	outputFile := inputFile + ".svg"
	cmd := exec.Command("mmdc", "-i", inputFile, outputFile)
	cmdOutput, err := cmd.CombinedOutput()

	defer func() {
		_ = os.Remove(outputFile)
	}()
	if err != nil {
		log.Printf("Unable to execute mermaid: %v\n\n%s\n", err, cmdOutput)
		return nil, err
	}
	if len(cmdOutput) > 0 {
		s := string(cmdOutput)
		log.Printf("Mermaid output:\n\n%s\n", s)
		return nil, errors.New(s)
	}

	output, err := ioutil.ReadFile(outputFile)
	if err != nil {
		log.Println("Unable to read back graph data", err)
		return nil, err
	}

	return cleanup(output), nil
}

// SVG is XHTML. If we include <br> in the output, we must use <br/>, etc.
func cleanup(s []byte) []byte {
	return []byte(strings.Replace(string(s), "<br>", "<br/>", -1))
}
