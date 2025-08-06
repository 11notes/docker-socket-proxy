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
	"strconv"
	"flag"
	"time"
)

var(
	proxy *httputil.ReverseProxy
	socket net.Listener
	wg sync.WaitGroup
	socketProxy string
	dockerSocket *net.Conn
)

func signals(){
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
	go func() {
		<-signalChannel
		os.Exit(0)
	}()
}

func prepareFileSystemDropPrivileges(){
	// unprivileged user
	proxyUID, err := strconv.Atoi(os.Getenv("SOCKET_PROXY_UID"))
	if err != nil {
		log.Fatalf("SOCKET_PROXY_UID must be a number %v", err)
	}
	proxyGID, err := strconv.Atoi(os.Getenv("SOCKET_PROXY_GID"))
	if err != nil {
		log.Fatalf("SOCKET_PROXY_GID must be a number %v", err)
	}
	proxyVolume := regexp.MustCompile(`/+$`).ReplaceAllString(os.Getenv("SOCKET_PROXY_VOLUME"), "")

	// chown file system for unprivileged user	
	if err := os.Chown(proxyVolume, proxyUID , proxyGID); err != nil {
		log.Fatalf("could not chown folder %s", proxyVolume, err)
	}

	// check docker socket permissions
	stat, err := os.Stat(os.Getenv("SOCKET_PROXY_DOCKER_SOCKET"))
	if err != nil {
		log.Fatalf("could not evaluate ownership of docker socket, permission issue %v", err)
	}
	if ownership, ok := stat.Sys().(*syscall.Stat_t); !ok {
		log.Fatalf("could not evaluate ownership of docker socket, permission issue %v", err)
	}else{
		if(int(ownership.Uid) != os.Getuid()){
			log.Fatalf("can’t access docker socket as UID %d owned by UID %d\nplease change the user setting in your compose to the correct UID/GID pair like this:\nservices:\n  socket-proxy:\n    user: \"%d:%d\"", os.Getuid(), ownership.Uid, ownership.Uid, ownership.Gid)
		}else{
			if(int(ownership.Gid) != os.Getgid()){
				log.Fatalf("can’t access docker socket as GID %d owned by GID %d\nplease change the user setting in your compose to the correct UID/GID pair like this:\nservices:\n  socket-proxy:\n    user: \"%d:%d\"", os.Getgid(), ownership.Gid, os.Getuid(), ownership.Gid)
			}
		}
	}

	// drop privileges since only the proxy must access the socket as root and nothing else
	if err := syscall.Setgid(proxyGID); err != nil {
		log.Fatalf("could not set GID to %d %v", proxyGID, err)
	}

	if err := syscall.Setuid(proxyUID); err != nil {
		log.Fatalf("could not set UID to %d %v", proxyUID, err)
	}
}

func httpProxyBlockedPaths(url string) bool {
	// block paths that use GET but pose security risk
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
	if((method  == "GET" || method  == "HEAD") && !httpProxyBlockedPaths(url)){
		proxy.ServeHTTP(w, r)
	}else{
		log.Printf("blocked: %s %s", method, url)
		http.Error(w, "", http.StatusForbidden)  
	}
}

func main(){
	// set socket proxy file path
	socketProxy = regexp.MustCompile(`/+$`).ReplaceAllString(os.Getenv("SOCKET_PROXY_VOLUME"), "") + "/docker.sock"

	// check for command line flags
	healthCheckFlag := flag.Bool("healthcheck", false, "just run healthcheck")
	flag.Parse()

	if(*healthCheckFlag){
		// only run healthcheck
		_, err := net.Dial("unix", socketProxy)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}else{
		log.Println("starting socket-proxy v" + os.Getenv("APP_VERSION"))
		// setup signal handler
		signals()

		// setup proxy to docker socket as root
		keepAlive, err := time.ParseDuration(os.Getenv("SOCKET_PROXY_KEEPALIVE"))
		if err != nil {
			log.Fatalf("%s not a valid time format: %s", os.Getenv("SOCKET_PROXY_KEEPALIVE"), err)
		}
		timeout, err := time.ParseDuration(os.Getenv("SOCKET_PROXY_TIMEOUT"))
		if err != nil {
			log.Fatalf("%s not a valid time format: %s", os.Getenv("SOCKET_PROXY_TIMEOUT"), err)
		}
		docketSockerDialer := &net.Dialer{KeepAlive: keepAlive, Timeout: timeout}
		dockerSocket, err := docketSockerDialer.Dial("unix", os.Getenv("SOCKET_PROXY_DOCKER_SOCKET"))
		if err != nil {
			log.Fatalf("could not access docker socket %v", err)
		}
		localhost, _ := url.Parse("http://localhost")
		proxy = httputil.NewSingleHostReverseProxy(localhost)
		proxy.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string)(net.Conn, error){
				dockerSocket, err = docketSockerDialer.Dial("unix", os.Getenv("SOCKET_PROXY_DOCKER_SOCKET"))
				if err != nil {
					log.Fatalf("could not access docker socket %v", err)
				}
				return dockerSocket, err
			},
		}

		// prepare the file system and drop privileges to UID/GID
		prepareFileSystemDropPrivileges()

		// setup unix to socket proxy
		unixServer := &http.Server{
			Handler: http.HandlerFunc(httpProxy),
		}
		os.Remove(socketProxy)
		unix, err := net.Listen("unix", socketProxy)
		if err != nil {
			log.Fatalf("could not start unix socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			log.Println("starting proxy UNIX socket ...")
			if err := unixServer.Serve(unix); err != nil {
				log.Fatalf("could not start unix socket %v", err)
			}
		}()

		// setup http to socket proxy
		httpServer := &http.Server{
			Handler: http.HandlerFunc(httpProxy),
		}

		tcp, err := net.Listen("tcp", "0.0.0.0:2375")
		if err != nil {
			log.Fatalf("could not start tcp socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			log.Println("starting proxy TCP socket ...")
			if err := httpServer.Serve(tcp); err != nil {
				log.Fatalf("could not start tcp socket %v", err)
			}
		}()

		// try to access the socket proxy
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, "http://localhost:2375/version", nil)
		if err != nil {
			log.Fatalf("could not create HTTP request %v", err)
		}
		res, err := client.Do(req)
		if err != nil {
			log.Fatalf("could not proxy to docker socket %v", err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			log.Fatalf("could not proxy to docker socket %v", err)
		}
		log.Println("proxy connection to docker socket established")

		wg.Wait()
	}
}