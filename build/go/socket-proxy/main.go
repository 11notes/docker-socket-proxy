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

	"github.com/11notes/go-eleven"
)

var(
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
	httpSocket string = os.Getenv("SOCKET_PROXY_HTTP_LISTEN_IP")
	containersOnlyEnv string = os.Getenv("SOCKET_PROXY_CONTAINERS_ONLY")
	containersOnly bool
	containerActionPattern = regexp.MustCompile(`(?i)^/(v[0-9]+\.[0-9]+/)?containers/[^/]+/(start|stop)$`)
	dockerReachable bool = true
)

func init(){
	if v, err := strconv.ParseBool(containersOnlyEnv); err == nil{
		containersOnly = v
	}
}

func prepareFileSystemDropPrivileges(){
	// unprivileged user
	proxyUID, err := strconv.Atoi(uid)
	if err != nil {
		eleven.LogFatal("SOCKET_PROXY_UID must be a number %v", err)
	}
	proxyGID, err := strconv.Atoi(gid)
	if err != nil {
		eleven.LogFatal("SOCKET_PROXY_GID must be a number %v", err)
	}
	proxyVolume := regexp.MustCompile(`/+$`).ReplaceAllString(volume, "")

	// chown file system for unprivileged user	
	if err := os.Chown(proxyVolume, proxyUID , proxyGID); err != nil {
		eleven.LogFatal("could not chown folder %s", proxyVolume, err)
	}

	// check docker socket permissions
	stat, err := os.Stat(dockerSocket)
	if err != nil {
		eleven.LogFatal("could not evaluate ownership of docker socket, permission issue %v", err)
	}
	if ownership, ok := stat.Sys().(*syscall.Stat_t); !ok {
		eleven.LogFatal("could not evaluate ownership of docker socket, permission issue %v", err)
	}else{
		if(int(ownership.Uid) != os.Getuid()){
			eleven.LogFatal("can’t access docker socket as UID %d owned by UID %d. Please change the user setting in your compose to the correct UID/GID pair like this >> user: %d:%d", os.Getuid(), ownership.Uid, ownership.Uid, ownership.Gid)
		}else{
			if(int(ownership.Gid) != os.Getgid()){
				eleven.LogFatal("can’t access docker socket as GID %d owned by GID %d. Please change the user setting in your compose to the correct UID/GID pair like this >> user: %d:%d", os.Getgid(), ownership.Gid, os.Getuid(), ownership.Gid)
			}
		}
	}

	// drop privileges since only the proxy must access the socket as root and nothing else
	if err := syscall.Setgid(proxyGID); err != nil {
		eleven.LogFatal("could not set GID to %d %v", proxyGID, err)
	}

	if err := syscall.Setuid(proxyUID); err != nil {
		eleven.LogFatal("could not set UID to %d %v", proxyUID, err)
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
		regexp.MustCompile(`(?i)images/get(/|)$`), // could exfil container data
	}

	for _, pattern := range blockedPatterns {
		if pattern.MatchString(url) {
			return true
		}
	}
	return false
}

func httpProxyAllowed(method string, path string, fullURL string) bool{
	// deny everything or only allow container start/stop
	if containersOnly {
		return method == http.MethodPost && containerActionPattern.MatchString(path)
	}
	return (method == http.MethodGet || method == http.MethodHead) && !httpProxyBlockedPaths(fullURL)
}

func httpProxy(w http.ResponseWriter, r *http.Request){
	method := r.Method
	fullURL := r.URL.String()
	path := r.URL.Path
	if httpProxyAllowed(method, path, fullURL){
		proxy.ServeHTTP(w, r)
	}else{
		eleven.Log("INF", "blocked: %s %s", method, fullURL)
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
		eleven.Log("DBG", "health check successfully")
	}
}

func dockerVersionRequest() *http.Request {
	// docker API check
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:2375/version", nil)
	if err != nil {
		eleven.LogFatal("could not create HTTP request %v", err)
	}
	return req
}

func waitForDockerSocket(client *http.Client){
	// wait for docker socket ready status
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	for {
		res, err := client.Do(dockerVersionRequest())
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				eleven.Log("INF", "docker socket is reachable")
				return
			}
			eleven.Log("WRN", "docker socket returned status %d, retrying in %s ...", res.StatusCode, backoff)
		}else{
			eleven.Log("WRN", "could not reach docker socket yet (%v), retrying in %s ...", err, backoff)
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func monitorDockerConnection(client *http.Client){
	// monitor docker socket status
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		res, err := client.Do(dockerVersionRequest())
		reachable := err == nil && res != nil && res.StatusCode == http.StatusOK
		if res != nil {
			res.Body.Close()
		}
		if reachable != dockerReachable {
			if reachable {
				eleven.Log("INF", "docker socket is reachable again")
			}else{
				eleven.Log("WRN", "docker socket became unreachable (docker service restarting?), will keep retrying in the background")
			}
			dockerReachable = reachable
		}
	}
}

func main(){
	// set socket proxy file path
	socketProxy = regexp.MustCompile(`/+$`).ReplaceAllString(volume, "") + "/docker.sock"

	if(eleven.Util.CommandLineArgumentExists("--healthcheck")){
		// only run healthcheck
		healthcheck(true)
	}else{
		// start app
		eleven.Log("START", "")
		if(containersOnly){
			eleven.Log("INF", "containers-only mode enabled: only starting/stopping containers is allowed")
		}

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
			eleven.LogFatal("%s not a valid time format: %s", keepAlive, err)
		}
		timeout, err := time.ParseDuration(timeout)
		if err != nil {
			eleven.LogFatal("%s not a valid time format: %s", timeout, err)
		}
		localhost, _ := url.Parse("http://127.0.0.1")
		proxy = httputil.NewSingleHostReverseProxy(localhost)
		docketSockerDialer := &net.Dialer{KeepAlive: keepAlive, Timeout: timeout}
		proxy.Transport = &http.Transport{
			DialContext:func(_ context.Context, _, _ string)(net.Conn, error){
				return(docketSockerDialer.Dial("unix", dockerSocket))
			},
			DisableKeepAlives:true,
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
			eleven.LogFatal("could not start unix socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			eleven.Log("INF", "starting proxy UNIX socket on " + socketProxy + " ...")
			if err := unixServer.Serve(unix); err != nil {
				eleven.LogFatal("could not start unix socket %v", err)
			}
		}()

		// setup http to socket proxy
		httpServer := &http.Server{
			Handler: http.HandlerFunc(httpProxy),
		}

		tcp, err := net.Listen("tcp", httpSocket + ":2375")
		if err != nil {
			eleven.LogFatal("could not start tcp socket %v", err)
		}
		wg.Add(1)
		go func(){
			defer wg.Done()
			eleven.Log("INF", "starting proxy TCP socket on " + httpSocket + "...")
			if err := httpServer.Serve(tcp); err != nil {
				eleven.LogFatal("could not start tcp socket %v", err)
			}
		}()

		// wait for docker to be reachable
		client := &http.Client{}
		waitForDockerSocket(client)

		// keep watching docker connectivity
		go monitorDockerConnection(client)

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