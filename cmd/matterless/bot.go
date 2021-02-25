package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
)

//go:embed HELP.md
var helpText string

// Bot represent the meeting bot object
type Bot struct {
	userCache    map[string]*model.User
	channelCache map[string]*model.Channel
	mmClient     *model.Client4
	wsClient     *model.WebSocketClient
	botUser      *model.User
}

// NewBot creates a new instance of the bot event listener
func NewBot(url, wsURL, token string) (*Bot, error) {
	mb := &Bot{
		userCache:    map[string]*model.User{},
		channelCache: map[string]*model.Channel{},
		mmClient:     model.NewAPIv4Client(url),
	}
	mb.mmClient.SetOAuthToken(token)

	var err *model.AppError
	mb.wsClient, err = model.NewWebSocketClient4(wsURL, token)
	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	var resp *model.Response
	mb.botUser, resp = mb.mmClient.GetMe("")
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(resp.Error, "Could not get bot account")
	}

	return mb, nil
}

// Listen listens and handles incoming messages until the socket disconnects
func (mb *Bot) Listen() error {
	err := mb.wsClient.Connect()

	if err != nil {
		return err
	}

	mb.wsClient.Listen()

	for evt := range mb.wsClient.EventChannel {
		log.Debug("Received event", evt)

		if evt.EventType() == "posted" {
			mb.handlePosted(evt)
		}
	}
	return nil
}

func (mb *Bot) lookupUser(userID string) *model.User {
	user := mb.userCache[userID]
	if user != nil {
		return user
	} else {
		mb.userCache[userID], _ = mb.mmClient.GetUser(userID, "")
		return mb.userCache[userID]
	}
}

func (mb *Bot) lookupChannel(channelID string) *model.Channel {
	channel := mb.channelCache[channelID]
	if channel != nil {
		return channel
	} else {
		mb.channelCache[channelID], _ = mb.mmClient.GetChannel(channelID, "")
		return mb.channelCache[channelID]
	}
}

func (mb *Bot) handleDefine(post *model.Post) {
}

func (mb *Bot) handleDirect(post *model.Post, channel *model.Channel) {
	words := strings.Split(post.Message, " ")
	switch words[0] {
	case "define":
		mb.handleDefine(post)
	case "ping":
		_, resp := mb.mmClient.SaveReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "ping_pong",
		})
		if resp.StatusCode != 200 {
			fmt.Printf("Failed to respond to ping: %+v", resp)
		}
	case "help":
		_, resp := mb.mmClient.CreatePost(&model.Post{
			ParentId:  post.Id,
			RootId:    post.Id,
			Message:   helpText,
			ChannelId: post.ChannelId,
		})
		if resp.StatusCode != http.StatusCreated {
			fmt.Printf("Failed to respond: %+v", resp)
		}
	}
}

func (mb *Bot) handleChannel(post *model.Post, channel *model.Channel) {
}

func (mb *Bot) handlePosted(evt *model.WebSocketEvent) {
	post := model.Post{}
	err := json.Unmarshal([]byte(evt.Data["post"].(string)), &post)
	if err != nil {
		log.Error("Could not unmarshall post", err)
		return
	}
	channel := mb.lookupChannel(post.ChannelId)
	if channel.Type == model.CHANNEL_DIRECT {
		mb.handleDirect(&post, channel)
	} else if channel.Type == model.CHANNEL_GROUP {
		mb.handleChannel(&post, channel)
	}

	log.Debug("Here is the post", post)
}
