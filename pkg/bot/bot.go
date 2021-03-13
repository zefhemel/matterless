package bot

import (
	_ "embed"
	"encoding/json"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/definition"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/eventsource"
)

//go:embed HELP.md
var helpText string

// MatterlessBot represent the meeting bot object
type MatterlessBot struct {
	botSource        *eventsource.BotSource
	userCache        map[string]*model.User
	channelCache     map[string]*model.Channel
	channelNameCache map[string]*model.Channel
	team             *model.Team
	botUser          *model.User

	userApps map[string]*application.Application
}

// NewBot creates a new instance of the bot event listener
func NewBot(url, adminToken string) (*MatterlessBot, error) {
	mb := &MatterlessBot{
		userCache:        map[string]*model.User{},
		channelCache:     map[string]*model.Channel{},
		channelNameCache: map[string]*model.Channel{},
		userApps:         map[string]*application.Application{},
	}
	adminClient := model.NewAPIv4Client(url)
	adminClient.SetOAuthToken(adminToken)

	var err error
	mb.botSource, err = eventsource.NewBotSource(adminClient, "matterless", &definition.BotDef{
		TeamNames:   []string{os.Getenv("team_name")},
		Username:    "matterless",
		DisplayName: "Matterless",
		Description: "Matterless Bot",
		Events: map[string][]definition.FunctionID{
			"all": {"CatchAll"},
		},
	}, func(name definition.FunctionID, event interface{}) interface{} {
		log.Debug("Received event", event)
		wsEvent := event.(*model.WebSocketEvent)

		if wsEvent.EventType() == "posted" || wsEvent.EventType() == "post_edited" {
			mb.handlePosted(wsEvent)
		}
		return nil
	})

	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	var resp *model.Response
	mb.team, resp = adminClient.GetTeamByName(os.Getenv("team_name"), "")
	logAPIResponse(resp, "get bot team")

	mb.botUser, resp = mb.botSource.BotUserClient.GetMe("")
	logAPIResponse(resp, "get bot user")

	return mb, nil
}

// Start listens and handles incoming messages until the socket disconnects
func (mb *MatterlessBot) Start() error {
	err := mb.botSource.Start()

	if err != nil {
		return err
	}

	mb.loadDeclarations()

	return nil
}

func (mb *MatterlessBot) lookupUser(userID string) *model.User {
	user := mb.userCache[userID]
	if user != nil {
		return user
	} else {
		mb.userCache[userID], _ = mb.botSource.BotUserClient.GetUser(userID, "")
		return mb.userCache[userID]
	}
}

func (mb *MatterlessBot) lookupChannel(channelID string) *model.Channel {
	channel := mb.channelCache[channelID]
	if channel != nil {
		return channel
	} else {
		mb.channelCache[channelID], _ = mb.botSource.BotUserClient.GetChannel(channelID, "")
		return mb.channelCache[channelID]
	}
}

// lookupChannelByName returns nil in case channel is not found
func (mb *MatterlessBot) lookupChannelByName(name string, teamID string) *model.Channel {
	channel := mb.channelNameCache[name]
	if channel != nil {
		return channel
	} else {
		mb.channelNameCache[name], _ = mb.botSource.BotUserClient.GetChannelByName(name, teamID, "")
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
	_, resp := mb.botSource.BotUserClient.CreatePost(&model.Post{
		ParentId:  post.Id,
		RootId:    post.Id,
		Message:   message,
		ChannelId: post.ChannelId,
	})
	logAPIResponse(resp, "replying to message")
}

func (mb *MatterlessBot) ensureReply(post *model.Post, message string) {
	pl, _ := mb.botSource.BotUserClient.GetPostThread(post.Id, "")
	for _, p := range pl.Posts {
		if p.UserId != mb.botUser.Id {
			continue
		}
		if p.ParentId == post.Id {
			p.Message = message
			_, resp := mb.botSource.BotUserClient.UpdatePost(p.Id, p)
			logAPIResponse(resp, "updating reply")
			return
		}
	}
	// No reply yet, create a new one
	mb.replyToPost(post, message)
}

func (mb *MatterlessBot) handleDirect(post *model.Post) {
	client := mb.botSource.BotUserClient
	if post.Message[0] == '#' {
		userApp, ok := mb.userApps[post.UserId]
		if !ok {
			userApp = application.NewApplication(client, func(kind, message string) {
				mb.postFunctionLog(post.UserId, kind, message)
			})
		}
		mb.userApps[post.UserId] = userApp
		err := userApp.Eval(post.Message)
		if err != nil {
			apiDelay()
			_, resp := client.DeleteReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "white_check_mark",
			})
			logAPIResponse(resp, "delete declaration reaction")
			apiDelay()
			_, resp = client.SaveReaction(&model.Reaction{
				UserId:    mb.botUser.Id,
				PostId:    post.Id,
				EmojiName: "stop_sign",
			})
			logAPIResponse(resp, "set declaration reaction")
			mb.ensureReply(post, err.Error())
			return
		}

		apiDelay()
		_, resp := client.DeleteReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "stop_sign",
		})
		logAPIResponse(resp, "delete declaration reaction")
		apiDelay()
		mb.ensureReply(post, "All good :thumbsup:")
		_, resp = client.SaveReaction(&model.Reaction{
			UserId:    mb.botUser.Id,
			PostId:    post.Id,
			EmojiName: "white_check_mark",
		})
		logAPIResponse(resp, "save declaration reaction")
	} else {
		switch post.Message {
		case "ping":
			_, resp := client.SaveReaction(&model.Reaction{
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
	client := mb.botSource.BotUserClient
	if p.Type == model.POST_LEAVE_CHANNEL {
		members, resp := client.GetChannelMembers(channel.Id, 0, 5, "")
		if resp.Error != nil {
			logAPIResponse(resp, "getting channel members")
			return
		}
		if len(*members) == 1 {
			// Just matterless bot left, close the channel
			success, resp := client.DeleteChannel(channel.Id)
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
	client := mb.botSource.BotUserClient
	channels, resp := client.GetChannelsForTeamForUser(mb.team.Id, "me", "")
	logAPIResponse(resp, "load all channels")
	for _, ch := range channels {
		if ch.Type == model.CHANNEL_DIRECT {
			log.Debugf("Processing direct channel: %+v", ch)
			posts, resp := client.GetPostsForChannel(ch.Id, 0, 100, "")
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
