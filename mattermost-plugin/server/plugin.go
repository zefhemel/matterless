package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/v6/plugin"

	log "github.com/sirupsen/logrus"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// Matterless stuff
	config    *config.Config
	container *application.Container
}

func (p *Plugin) OnActivate() error {
	log.SetLevel(log.DebugLevel)
	return nil
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	if userID != "" {
		user, err := p.API.GetUser(userID)
		if err != nil {
			mlog.Error("Error in authenticated user lookup", mlog.Err(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if user.IsSystemAdmin() {
			r.Header.Set("Authorization", fmt.Sprintf("bearer %s", p.config.AdminToken))
		}
	}
	mlog.Info(fmt.Sprintf("Got HTTP request: %s: %s Headers: %+v", r.Method, r.URL, r.Header))

	// TODO: Less naive URL filter
	if strings.HasSuffix(r.URL.String(), "/_events") {
		p.wsProxy(w, r)
	} else {
		p.httpProxy(w, r)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (p *Plugin) wsProxy(w http.ResponseWriter, r *http.Request) {
	wsUrl, _ := url.Parse(fmt.Sprintf("ws://localhost:%d%s", p.config.APIBindPort, r.URL))
	wsProxy := websocketproxy.NewProxy(wsUrl)
	//wsProxy.Director
	wsProxy.ServeHTTP(w, r)
}

func (p *Plugin) httpProxy(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	req, err := http.NewRequest(r.Method, fmt.Sprintf("http://localhost:%d%s", p.config.APIBindPort, r.URL), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy error: %s", err), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy error: %s", err), http.StatusInternalServerError)
		return
	}
	for k, vs := range res.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)
	_, err = io.Copy(w, res.Body)
	if err != nil {
		mlog.Error("Error proxying", mlog.Err(err))
	}
	res.Body.Close()
}
