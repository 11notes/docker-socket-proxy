package main

import(
	"log"
	"net"
	"net/http"
	"net/url"
	"net/http/httputil"
	"context"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"regexp"
)

var(
	proxy *httputil.ReverseProxy
	socket net.Listener
	wg sync.WaitGroup
)

func signals(){
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
	go func() {
		<-signalChannel
		os.Exit(0)
	}()
}

func httpProxyBlockedPaths(url string) bool {
	blockedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)containers/\S+/attach/ws.*`), // could attach to stdin via web socket and issue command inside the container
		regexp.MustCompile(`(?i)containers/\S+/export.*`), // could exfil container data
		regexp.MustCompile(`(?i)containers/\S+/archive.*`), // could exfil container data
		regexp.MustCompile(`(?i)secrets.*`), // could exfil credentials
		regexp.MustCompile(`(?i)configs.*`), // could exfil credentials
		regexp.MustCompile(`(?i)swarm/unlockkey.*`), // could exfil credentials
		regexp.MustCompile(`(?i)images/get.*`), // could exfil container data
	}

	for _, pattern := range blockedPatterns {
		if pattern.MatchString(url) {
			return true
		}
	}
	return false
}

func httpProxy(w http.ResponseWriter, r *http.Request){
	method := r.Method
	url := r.URL.String()
	if(method  == "GET" && !httpProxyBlockedPaths(url)){
		proxy.ServeHTTP(w, r)
	}else{
		log.Printf("blocked: %s %s", method, url)
		http.Error(w, "", http.StatusForbidden)  
	}
}

func main(){
	signals()

	// setup proxy to docker socket
	localhost, _ := url.Parse("http://localhost")
	proxy = httputil.NewSingleHostReverseProxy(localhost)
	proxy.Transport = &http.Transport{
		DialContext: func(_ context.Context, _, _ string)(net.Conn, error){
			return net.Dial("unix", os.Getenv("SOCKET_PROXY_DOCKER_SOCKET"))
		},
	}

	// drop privileges since only the proxy must access the socket as root and nothing else
	if err := syscall.Setgid(1000); err != nil {
		log.Fatalf("could not set GID to 1000 %v", err)
	}

	if err := syscall.Setuid(1000); err != nil {
		log.Fatalf("could not set UID to 1000 %v", err)
	}

	wg.Add(2)

	// setup unix to socket proxy
	serverUnix := &http.Server{
		Handler: http.HandlerFunc(httpProxy),
	}

	os.Remove(os.Getenv("SOCKET_PROXY"))
	unix, _ := net.Listen("unix", os.Getenv("SOCKET_PROXY"))
	go func(){
		defer wg.Done()
		if err := serverUnix.Serve(unix); err != nil {
			log.Fatalf("could not start unix socket %v", err)
		}
	}()

	// setup http to socket proxy
	httpServer := &http.Server{
		Handler: http.HandlerFunc(httpProxy),
	}

	tcp, _ := net.Listen("tcp", "0.0.0.0:8080")
	go func(){
		defer wg.Done()
		if err := httpServer.Serve(tcp); err != nil {
			log.Fatalf("could not start tcp socket %v", err)
		}
	}()

	wg.Wait()
}