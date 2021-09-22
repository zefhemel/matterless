package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
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
	return sandbox.EnsureDeno(p.config)
}

func (p *Plugin) OnDeactivate() error {
	p.API.LogInfo("Deactivating...")
	if p.container != nil {
		p.container.Close()
	}
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
	//p.API.LogInfo("Now proxying Websocket")
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("websocket error: %s", err), http.StatusBadRequest)
		return
	}
	proxyConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d%s", p.config.APIBindPort, r.URL), nil)
	if err != nil {
		p.API.LogError("websocket proxy error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
		return
	}

	defer clientConn.Close()
	defer proxyConn.Close()

	go func() {
		// Pass-through proxy from proxyConn up to the clientConn
		for {
			messageType, msg, err := proxyConn.ReadMessage()
			if err != nil {
				// Ignore close errors
				if _, ok := err.(*websocket.CloseError); !ok {
					log.Errorf("Websocket error: %s", err)
				}
				return
			}
			if err := clientConn.WriteMessage(messageType, msg); err != nil {
				p.API.LogError("WS proxy error", "err", err)
			}
		}
	}()

messageLoop:
	for {
		messageType, msg, err := clientConn.ReadMessage()
		if err != nil {
			// Ignore close errors
			if _, ok := err.(*websocket.CloseError); !ok {
				log.Errorf("Websocket error: %s", err)
			}
			return
		}
		// We are going to intercept authentication messages and swap in our own check and token
		if messageType == websocket.TextMessage {
			var clientMessage application.WSEventClientMessage
			if err := json.Unmarshal(msg, &clientMessage); err != nil {
				p.API.LogError("Could not parse websocket message", "error", err)
				continue messageLoop
			}
			if clientMessage.Type == "authenticate" {
				//p.API.LogInfo("Need to authenticate with token", "token", clientMessage.Token)
				siteUrl := *p.API.GetConfig().ServiceSettings.SiteURL
				client := model.NewAPIv4Client(siteUrl)
				client.SetToken(clientMessage.Token)
				authUser, _, err := client.GetMe("")
				if err != nil {
					p.API.LogError("Failed to authenticate", "err", err)
					continue
				}
				if authUser.IsInRole("system_admin") {
					// Admin!
					//p.API.LogInfo("An admin authenticated!")
					if err := proxyConn.WriteMessage(messageType, util.MustJsonByteSlice(application.WSEventClientMessage{
						Type:  "authenticate",
						Token: p.configuration.AdminToken,
					})); err != nil {
						p.API.LogError("Could not proxy authenticate message", "err", err)
					}
				}
				continue
			}
		}
		//p.API.LogInfo("Proxying message")
		proxyConn.WriteMessage(messageType, msg)
	}
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
