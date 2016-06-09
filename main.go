package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

var (
	addr = flag.String("addr", "0.0.0.0", "Server listen ip")
	port = flag.Int("port", 8123, "Server listen port")
	cert = flag.String("cert", "", "TLS certificate")
	key  = flag.String("key", "", "TLS certificate private key")

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
	getUrl := r.URL.Query().Get("url")
	if getUrl == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`Need the "url" query parameter`))
		return
	}
	urlObj, err := url.Parse(getUrl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	request, err := http.NewRequest(r.Method, getUrl, nil)
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

	response, err := httpClient.Do(request)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	defer response.Body.Close()

	wHeader := w.Header()
	wHeader["Content-Disposition"] = []string{"attachment; filename=\"" + getFilenameFromPath(urlObj.Path) + "\""}
	for key, value := range response.Header {
		wHeader[key] = value
	}
	w.WriteHeader(response.StatusCode)
	buffer := bufferPool.Get().([]byte)
	defer bufferPool.Put(buffer)
	for {
		n, err := response.Body.Read(buffer)
		if err != nil {
			break
		}
		w.Write(buffer[:n])
	}
}

func main() {
	flag.Parse()
	httpClient = &http.Client{}
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8*1024)
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	server := &http.Server{Addr: *addr + ":" + strconv.Itoa(*port), Handler: mux}

	if *cert != "" && *key != "" {
		log.Println("Starting TLS HTTP Server")
		log.Fatal(server.ListenAndServeTLS(*cert, *key))
	} else {
		log.Println("Starting HTTP Server")
		log.Fatal(server.ListenAndServe())
	}
}
