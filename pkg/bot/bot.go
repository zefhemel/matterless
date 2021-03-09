package bot

import (
	_ "embed"
	"encoding/json"
	"github.com/zefhemel/matterless/pkg/application"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/eventsource"
)

//go:embed HELP.md
var helpText string

// MatterlessBot represent the meeting bot object
type MatterlessBot struct {
	userCache        map[string]*model.User
	channelCache     map[string]*model.Channel
	channelNameCache map[string]*model.Channel
	team             *model.Team
	mmClient         *model.Client4
	eventSource      *eventsource.MatterMostSource
	botUser          *model.User

	userApps map[string]*application.Application
}

// NewBot creates a new instance of the bot event listener
func NewBot(url, token string) (*MatterlessBot, error) {
	mb := &MatterlessBot{
		userCache:        map[string]*model.User{},
		channelCache:     map[string]*model.Channel{},
		channelNameCache: map[string]*model.Channel{},
		mmClient:         model.NewAPIv4Client(url),
		userApps:         map[string]*application.Application{},
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

	teams, resp := mb.mmClient.GetTeamsForUser(mb.botUser.Id, "")
	logAPIResponse(resp, "get bot teams")
	mb.team = teams[0] // TODO Come up with something better

	return mb, nil
}

// Start listens and handles incoming messages until the socket disconnects
func (mb *MatterlessBot) Start() error {
	err := mb.eventSource.Start()

	if err != nil {
		return err
	}

	mb.loadDeclarations()

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

// lookupChannelByName returns nil in case channel is not found
func (mb *MatterlessBot) lookupChannelByName(name string, teamID string) *model.Channel {
	channel := mb.channelNameCache[name]
	if channel != nil {
		return channel
	} else {
		mb.channelNameCache[name], _ = mb.mmClient.GetChannelByName(name, teamID, "")
		return mb.channelNameCache[name]
	}
}

// It seems that without this, the WebUI (?) doesn't handle updates in order
func apiDelay() {
	time.Sleep(100 * time.Millisecond)
}

func logAPIResponse(resp *model.Response, action string) {
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
	logAPIResponse(resp, "replying to message")
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
			logAPIResponse(resp, "updating reply")
			return
		}
	}
	// No reply yet, create a new one
	mb.replyToPost(post, message)
}

func (mb *MatterlessBot) handleDirect(post *model.Post) {
	if post.Message[0] == '#' {
		userApp, ok := mb.userApps[post.UserId]
		if !ok {
			userApp = application.NewApplication(func(kind, message string) {
				mb.postFunctionLog(post.UserId, kind, message)
			})
		}
		mb.userApps[post.UserId] = userApp
		err := userApp.Eval(post.Message)
		if err != nil {
			apiDelay()
			_, resp := mb.mmClient.DeleteReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "white_check_mark",
			})
			logAPIResponse(resp, "delete declaration reaction")
			apiDelay()
			_, resp = mb.mmClient.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "stop_sign",
			})
			logAPIResponse(resp, "set declaration reaction")
			mb.ensureReply(post, err.Error())
			return
		}

		apiDelay()
		_, resp := mb.mmClient.DeleteReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "stop_sign",
		})
		logAPIResponse(resp, "delete declaration reaction")
		apiDelay()
		mb.ensureReply(post, "All good :thumbsup:")
		_, resp = mb.mmClient.SaveReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "white_check_mark",
		})
		logAPIResponse(resp, "save declaration reaction")
	} else {
		switch post.Message {
		case "ping":
			_, resp := mb.mmClient.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "ping_pong",
			})
			logAPIResponse(resp, "ping pong")
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
	switch channel.Type {
	case model.CHANNEL_DIRECT:
		mb.handleDirect(&post)
	case model.CHANNEL_PRIVATE:
		mb.handlePrivate(&post, channel)
	}
}

// Note: currently does nothing
func (mb *MatterlessBot) handlePrivate(p *model.Post, channel *model.Channel) {
	if strings.HasPrefix(channel.Name, "mls-bot-logs-") {
		// Log channel

		// TODO: Re-enable when figuring out how to recreate channels
		//mb.handleLogChannelLeaving(p, channel)
	}
}

// Note: not currently used
func (mb *MatterlessBot) handleLogChannelLeaving(p *model.Post, channel *model.Channel) {
	if p.Type == model.POST_LEAVE_CHANNEL {
		members, resp := mb.mmClient.GetChannelMembers(channel.Id, 0, 5, "")
		if resp.Error != nil {
			logAPIResponse(resp, "getting channel members")
			return
		}
		if len(*members) == 1 {
			// Just matterless bot left, close the channel
			success, resp := mb.mmClient.DeleteChannel(channel.Id)
			if !success {
				logAPIResponse(resp, "deleting log channel")
			} else {
				// Update caches
				delete(mb.channelCache, channel.Id)
				delete(mb.channelNameCache, channel.Name)
			}
			log.Info("Deleted channel ", channel.Name, "Success: ", success)
		}
	}
}

func (mb *MatterlessBot) loadDeclarations() {
	channels, resp := mb.mmClient.GetChannelsForTeamForUser(mb.team.Id, "me", "")
	logAPIResponse(resp, "load all channels")
	for _, ch := range channels {
		if ch.Type == model.CHANNEL_DIRECT {
			log.Debugf("Processing direct channel: %+v", ch)
			posts, resp := mb.mmClient.GetPostsForChannel(ch.Id, 0, 100, "")
			logAPIResponse(resp, "get posts for direct channel")
			// Note: posts are ordered in reverse-chronological order
			for _, postID := range posts.Order {
				post := posts.Posts[postID]
				if post.Message[0] == '#' {
					log.Debug("Found a declaration block", post.Message)
					mb.handleDirect(post)
					log.Debug("Loading at boot done for this channel")
					break
				}
			}
		}
	}
}
