package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type (
	RedirectedUrlMap map[*http.Request]*url.URL
)

var (
	addr    = flag.String("addr", "0.0.0.0", "Server listen ip")
	port    = flag.Int("port", 8123, "Server listen port")
	tlsPort = flag.Int("tls-port", 8124, "Server listen port for TLS")
	cert    = flag.String("cert", "", "TLS certificate")
	key     = flag.String("key", "", "TLS certificate private key")

	redirectedUrlMapLock sync.Mutex
	redirectedUrlMap     RedirectedUrlMap

	httpClient *http.Client
	bufferPool sync.Pool
)

func getFilenameFromPath(path string) string {
	v := strings.Split(path, "/")
	if len(v) > 1 {
		return v[len(v)-1]
	}
	return "index.html"
}

func handler(w http.ResponseWriter, r *http.Request) {
	theUrl := r.URL.Query().Get("url")
	base64EncodedUrl := r.URL.Query().Get("base64Url")
	if theUrl == "" {
		if base64EncodedUrl != "" {
			decoded, err := base64.StdEncoding.DecodeString(base64EncodedUrl)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`Failed to decode "base64Url" query parameter: ` + err.Error()))
				return
			}
			theUrl = string(decoded)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`Need the "url" or "base64Url" query parameter`))
			return
		}
	}
	urlObj, err := url.Parse(theUrl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	request, err := http.NewRequest(r.Method, theUrl, nil)
	if r.Body != nil {
		request.Body = r.Body
		if r.ContentLength > 0 {
			request.ContentLength = r.ContentLength
		}
	}
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	for key, value := range r.Header {
		if key != "Host" {
			request.Header[key] = value
		}
	}

	defer func() {
		redirectedUrlMapLock.Lock()
		delete(redirectedUrlMap, request)
		redirectedUrlMapLock.Unlock()
		if len(redirectedUrlMap) > 20 {
			log.Println("Warning: Too many items in |redirectedUrlMap|:", len(redirectedUrlMap), ", are objects leaking?")
		}
	}()

	response, err := httpClient.Do(request)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	defer response.Body.Close()

	finalUrl := urlObj
	redirectedUrlMapLock.Lock()
	redirectedUrl, exists := redirectedUrlMap[request]
	redirectedUrlMapLock.Unlock()
	if exists {
		log.Println("Redirect to:", redirectedUrl.String(), "from:", request.URL.String())
		finalUrl = redirectedUrl
	}

	wHeader := w.Header()
	wHeader["Content-Disposition"] = []string{"inline; filename=\"" + getFilenameFromPath(finalUrl.Path) + "\""}
	for key, value := range response.Header {
		wHeader[key] = value
	}
	w.WriteHeader(response.StatusCode)

	done := make(chan bool)
	redirectedBytes := int64(0)
	cancel := int32(0)
	go func() {
		buffer := bufferPool.Get().([]byte)
		defer bufferPool.Put(buffer)
		for atomic.LoadInt32(&cancel) == 0 {
			var wErr error = nil
			n, err := response.Body.Read(buffer)
			if atomic.LoadInt32(&cancel) > 0 {
				// The request has been canceled, so don't bother the response writer anymore.
				break
			}
			if n > 0 {
				n, wErr = w.Write(buffer[:n])
				atomic.AddInt64(&redirectedBytes, int64(n))
			}
			if err != nil {
				if err != io.EOF {
					log.Println("Read error for", request.URL.String(), ":", err.Error())
				}
				break
			}
			if wErr != nil {
				log.Println("Write error for", request.URL.String(), ":", wErr.Error())
				break
			}
		}
		done <- true
	}()

FORLOOP:
	for {
		const CheckInterval = 30
		select {
		case <-done:
			break FORLOOP
		case <-time.After(CheckInterval * time.Second):
			b := atomic.LoadInt64(&redirectedBytes)
			if b == 0 {
				log.Printf("Error: redirected 0 bytes in %d seconds", CheckInterval)
				atomic.StoreInt32(&cancel, 1)
				break FORLOOP
			} else {
				atomic.AddInt64(&redirectedBytes, -b)
			}
		}
	}
}

func main() {
	flag.Parse()

	redirectedUrlMapLock = sync.Mutex{}
	redirectedUrlMap = make(RedirectedUrlMap)

	httpClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			redirectedUrlMapLock.Lock()
			redirectedUrlMap[via[0]] = req.URL
			redirectedUrlMapLock.Unlock()
			return nil
		},
	}
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8*1024)
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	if *cert != "" && *key != "" {
		log.Println("Starting TLS HTTP Server")
		server := &http.Server{
			Addr:    *addr + ":" + strconv.Itoa(*tlsPort),
			Handler: mux,
		}
		log.Fatal(server.ListenAndServeTLS(*cert, *key))
	} else {
		log.Println("Starting HTTP Server")
		server := &http.Server{
			Addr:    *addr + ":" + strconv.Itoa(*port),
			Handler: mux,
		}
		log.Fatal(server.ListenAndServe())
	}
}
