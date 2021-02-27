package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/declaration"
	"github.com/zefhemel/matterless/pkg/eventsource"
	"github.com/zefhemel/matterless/pkg/interpreter"
	"github.com/zefhemel/matterless/pkg/sandbox"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
)

//go:embed HELP.md
var helpText string

// MatterlessBot represent the meeting bot object
type MatterlessBot struct {
	userCache    map[string]*model.User
	channelCache map[string]*model.Channel
	mmClient     *model.Client4
	eventSource  *eventsource.MatterMostSource
	botUser      *model.User
}

// NewBot creates a new instance of the bot event listener
func NewBot(url, token string) (*MatterlessBot, error) {
	mb := &MatterlessBot{
		userCache:    map[string]*model.User{},
		channelCache: map[string]*model.Channel{},
		mmClient:     model.NewAPIv4Client(url),
	}
	mb.mmClient.SetOAuthToken(token)

	var err error
	mb.eventSource, err = eventsource.NewMatterMostSource(url, token)
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
func (mb *MatterlessBot) Start() error {
	err := mb.eventSource.Start()

	if err != nil {
		return err
	}

	for evt := range mb.eventSource.Events() {
		log.Debug("Received event", evt)

		wsEvent := evt.(*model.WebSocketEvent)

		if wsEvent.EventType() == "posted" || wsEvent.EventType() == "post_edited" {
			mb.handlePosted(wsEvent)
		}
	}
	return nil
}

func (mb *MatterlessBot) lookupUser(userID string) *model.User {
	user := mb.userCache[userID]
	if user != nil {
		return user
	} else {
		mb.userCache[userID], _ = mb.mmClient.GetUser(userID, "")
		return mb.userCache[userID]
	}
}

func (mb *MatterlessBot) lookupChannel(channelID string) *model.Channel {
	channel := mb.channelCache[channelID]
	if channel != nil {
		return channel
	} else {
		mb.channelCache[channelID], _ = mb.mmClient.GetChannel(channelID, "")
		return mb.channelCache[channelID]
	}
}

// It seems that without this, the WebUI (?) doesn't handle updates in order
func apiDelay() {
	time.Sleep(100 * time.Millisecond)
}

func logApiResponse(resp *model.Response, action string) {
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		// Something's not right
		log.Errorf("HTTP Error status %d during %s: %s", resp.StatusCode, action, resp.Error.Message)
	}
}

func (mb *MatterlessBot) replyToPost(post *model.Post, message string) {
	_, resp := mb.mmClient.CreatePost(&model.Post{
		ParentId:  post.Id,
		RootId:    post.Id,
		Message:   message,
		ChannelId: post.ChannelId,
	})
	logApiResponse(resp, "replying to message")
}

func (mb *MatterlessBot) ensureReply(post *model.Post, message string) {
	pl, _ := mb.mmClient.GetPostThread(post.Id, "")
	for _, p := range pl.Posts {
		if p.UserId != mb.botUser.Id {
			continue
		}
		if p.ParentId == post.Id {
			p.Message = message
			_, resp := mb.mmClient.UpdatePost(p.Id, p)
			logApiResponse(resp, "updating reply")
			return
		}
	}
	// No reply yet, create a new one
	mb.replyToPost(post, message)
}

func (mb *MatterlessBot) handleDirect(post *model.Post, channel *model.Channel) {
	if post.Message[0] == '#' {
		// Declarations
		decls, err := declaration.Parse([]string{post.Message})
		if err != nil {
			log.Error(err)
			return
		}
		results := declaration.Check(decls)
		if results.String() != "" {
			// Error while checking
			apiDelay()
			_, resp := mb.mmClient.DeleteReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "white_check_mark",
			})
			logApiResponse(resp, "delete declaration reaction")
			apiDelay()
			_, resp = mb.mmClient.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "stop_sign",
			})
			logApiResponse(resp, "set declaration reaction")
			mb.ensureReply(post, fmt.Sprintf("Errors :thumbsdown:\n\n```\n%s\n```", results.String()))
			return
		}
		nodeSandbox := sandbox.NewNodeSandbox("node")
		testResults := interpreter.TestDeclarations(decls, nodeSandbox)
		if testResults.String() != "" {
			// Error while running
			apiDelay()
			_, resp := mb.mmClient.DeleteReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "white_check_mark",
			})
			logApiResponse(resp, "delete declaration reaction")
			apiDelay()
			_, resp = mb.mmClient.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "stop_sign",
			})
			logApiResponse(resp, "set declaration reaction")
			mb.ensureReply(post, fmt.Sprintf("Errors :thumbsdown:\n\n```\n%s\n```", testResults.String()))
			return
		}
		apiDelay()
		_, resp := mb.mmClient.DeleteReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "stop_sign",
		})
		logApiResponse(resp, "delete declaration reaction")
		apiDelay()
		mb.ensureReply(post, "All good :thumbsup:")
		_, resp = mb.mmClient.SaveReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "white_check_mark",
		})
		logApiResponse(resp, "save declaration reaction")
	} else {
		switch post.Message {
		case "ping":
			_, resp := mb.mmClient.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "ping_pong",
			})
			logApiResponse(resp, "ping pong")
		case "help":
			mb.replyToPost(post, helpText)
		}
	}
}

func (mb *MatterlessBot) handlePosted(evt *model.WebSocketEvent) {
	post := model.Post{}
	err := json.Unmarshal([]byte(evt.Data["post"].(string)), &post)
	if err != nil {
		log.Error("Could not unmarshall post", err)
		return
	}
	channel := mb.lookupChannel(post.ChannelId)
	if channel.Type == model.CHANNEL_DIRECT {
		mb.handleDirect(&post, channel)
	}
}
