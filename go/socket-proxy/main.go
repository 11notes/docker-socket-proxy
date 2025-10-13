package main

import(
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
	"time"

	"github.com/11notes/go"
)

var(
	Eleven eleven.New = eleven.New{}
	proxy *httputil.ReverseProxy
	socket net.Listener
	wg sync.WaitGroup
	socketProxy string
	keepAlive string = os.Getenv("SOCKET_PROXY_KEEPALIVE")
	timeout string = os.Getenv("SOCKET_PROXY_TIMEOUT")
	uid string = os.Getenv("SOCKET_PROXY_UID")
	gid string = os.Getenv("SOCKET_PROXY_GID")
	volume string = os.Getenv("SOCKET_PROXY_VOLUME")
	dockerSocket string = os.Getenv("SOCKET_PROXY_DOCKER_SOCKET")
)

func prepareFileSystemDropPrivileges(){
	// unprivileged user
	proxyUID, err := strconv.Atoi(uid)
	if err != nil {
		Eleven.LogFatal("SOCKET_PROXY_UID must be a number %v", err)
	}
	proxyGID, err := strconv.Atoi(gid)
	if err != nil {
		Eleven.LogFatal("SOCKET_PROXY_GID must be a number %v", err)
	}
	proxyVolume := regexp.MustCompile(`/+$`).ReplaceAllString(volume, "")

	// chown file system for unprivileged user	
	if err := os.Chown(proxyVolume, proxyUID , proxyGID); err != nil {
		Eleven.LogFatal("could not chown folder %s", proxyVolume, err)
	}

	// check docker socket permissions
	stat, err := os.Stat(dockerSocket)
	if err != nil {
		Eleven.LogFatal("could not evaluate ownership of docker socket, permission issue %v", err)
	}
	if ownership, ok := stat.Sys().(*syscall.Stat_t); !ok {
		Eleven.LogFatal("could not evaluate ownership of docker socket, permission issue %v", err)
	}else{
		if(int(ownership.Uid) != os.Getuid()){
			Eleven.LogFatal("can’t access docker socket as UID %d owned by UID %d. Please change the user setting in your compose to the correct UID/GID pair like this >> user: %d:%d", os.Getuid(), ownership.Uid, ownership.Uid, ownership.Gid)
		}else{
			if(int(ownership.Gid) != os.Getgid()){
				Eleven.LogFatal("can’t access docker socket as GID %d owned by GID %d. Please change the user setting in your compose to the correct UID/GID pair like this >> user: %d:%d", os.Getgid(), ownership.Gid, os.Getuid(), ownership.Gid)
			}
		}
	}

	// drop privileges since only the proxy must access the socket as root and nothing else
	if err := syscall.Setgid(proxyGID); err != nil {
		Eleven.LogFatal("could not set GID to %d %v", proxyGID, err)
	}

	if err := syscall.Setuid(proxyUID); err != nil {
		Eleven.LogFatal("could not set UID to %d %v", proxyUID, err)
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
		Eleven.Log("INF", "blocked: %s %s", method, url)
		http.Error(w, "", http.StatusForbidden)  
	}
}

func healthcheck(exit bool){
	healthcheckSockerDialer := &net.Dialer{Timeout: 2*time.Second}
	socket, err := healthcheckSockerDialer.Dial("unix", socketProxy)
	if err != nil {
		os.Exit(1)
	}
	err = socket.Close()
	if err != nil {
		os.Exit(1)
	}
	if(exit){
		os.Exit(0)
	}else{
		Eleven.Log("DBG", "health check successfully")
	}
}

func main(){
	// set socket proxy file path
	socketProxy = regexp.MustCompile(`/+$`).ReplaceAllString(volume, "") + "/docker.sock"

	if(Eleven.Util.CommandLineArgumentExists("--healthcheck")){
		// only run healthcheck
		healthcheck(true)
	}else{
		// start app
		Eleven.Log("START", "")

		// setup signal handler
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT)
		go func(){
			<-signalChannel
			os.Exit(1)
		}()

		// setup proxy to docker socket as root
		keepAlive, err := time.ParseDuration(keepAlive)
		if err != nil {
			Eleven.LogFatal("%s not a valid time format: %s", keepAlive, err)
		}
		timeout, err := time.ParseDuration(timeout)
		if err != nil {
			Eleven.LogFatal("%s not a valid time format: %s", timeout, err)
		}
		localhost, _ := url.Parse("http://localhost")
		proxy = httputil.NewSingleHostReverseProxy(localhost)
		docketSockerDialer := &net.Dialer{KeepAlive: keepAlive, Timeout: timeout}
		proxy.Transport = &http.Transport{
			DialContext:func(_ context.Context, _, _ string)(net.Conn, error){
				return(docketSockerDialer.Dial("unix", dockerSocket))
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
			Eleven.LogFatal("could not start unix socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			Eleven.Log("INF", "starting proxy UNIX socket ...")
			if err := unixServer.Serve(unix); err != nil {
				Eleven.LogFatal("could not start unix socket %v", err)
			}
		}()

		// setup http to socket proxy
		httpServer := &http.Server{
			Handler: http.HandlerFunc(httpProxy),
		}

		tcp, err := net.Listen("tcp", "0.0.0.0:2375")
		if err != nil {
			Eleven.LogFatal("could not start tcp socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			Eleven.Log("INF", "starting proxy TCP socket ...")
			if err := httpServer.Serve(tcp); err != nil {
				Eleven.LogFatal("could not start tcp socket %v", err)
			}
		}()

		// try to access the socket proxy
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, "http://localhost:2375/version", nil)
		if err != nil {
			Eleven.LogFatal("could not create HTTP request %v", err)
		}
		res, err := client.Do(req)
		if err != nil {
			Eleven.LogFatal("could not proxy to docker socket %v", err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			Eleven.LogFatal("could not proxy to docker socket %v", err)
		}

		// set internal socket check
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
				case <-ticker.C:
					healthcheck(false)
			}
		}

		// wait for socket to get stopped
		wg.Wait()
	}
}