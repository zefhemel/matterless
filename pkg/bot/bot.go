package bot

import (
	_ "embed"
	"encoding/json"
	"github.com/zefhemel/matterless/pkg/application"
	"github.com/zefhemel/matterless/pkg/definition"
	"os"
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

	postApps    map[string]*application.Application // post_id -> app
	adminClient *model.Client4
}

// NewBot creates a new instance of the bot event listener
func NewBot(url, adminToken string) (*MatterlessBot, error) {
	mb := &MatterlessBot{
		userCache:        map[string]*model.User{},
		channelCache:     map[string]*model.Channel{},
		channelNameCache: map[string]*model.Channel{},
		postApps:         map[string]*application.Application{},
		adminClient:      model.NewAPIv4Client(url),
	}

	mb.adminClient.SetOAuthToken(adminToken)

	var err error
	mb.botSource, err = eventsource.NewBotSource(mb.adminClient, "matterless", &definition.BotDef{
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
		} else if wsEvent.EventType() == "post_deleted" {
			mb.handleDeleted(wsEvent)
		}
		return nil
	})

	if err != nil {
		log.Error("Connecting to websocket", err)
		return nil, err
	}

	var resp *model.Response
	mb.team, resp = mb.adminClient.GetTeamByName(os.Getenv("team_name"), "")
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

// :arrows_counterclockwise:

func (mb *MatterlessBot) setOnlyReaction(post *model.Post, emoji string) {
	client := mb.botSource.BotUserClient
	for _, reaction := range post.Metadata.Reactions {
		if reaction.UserId == mb.botUser.Id {
			log.Debug("Deleting reaction ", reaction.EmojiName)
			_, resp := client.DeleteReaction(reaction)
			logAPIResponse(resp, "delete reaction")
			apiDelay()
		}
	}
	newReaction := &model.Reaction{
		UserId:    mb.botUser.Id,
		PostId:    post.Id,
		EmojiName: emoji,
	}
	log.Debug("Adding reaction ", newReaction)
	_, resp := client.SaveReaction(newReaction)
	post.Metadata.Reactions = []*model.Reaction{newReaction}
	logAPIResponse(resp, "save reaction")
}

func (mb *MatterlessBot) handleDirect(post *model.Post) {
	client := mb.botSource.BotUserClient
	if post.Message[0] == '#' {
		postApp, ok := mb.postApps[post.Id]
		if !ok {
			postApp = application.NewApplication(mb.adminClient, func(kind, message string) {
				mb.postFunctionLog(post.UserId, kind, message)
			})
		}
		mb.postApps[post.Id] = postApp
		if postApp.CurrentCode() == post.Message {
			log.Debug("Code hasn't modified, skipping")
			return
		}
		// loading
		mb.setOnlyReaction(post, "arrows_counterclockwise")

		err := postApp.Eval(post.Message)
		if err != nil {
			mb.setOnlyReaction(post, "stop_sign")
			mb.ensureReply(post, err.Error())
			return
		}

		mb.setOnlyReaction(post, "white_check_mark")
		mb.ensureReply(post, "All good :thumbsup:")
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
				}
			}
			log.Debug("Loading at boot done for this channel")
		}
	}
}

func (mb *MatterlessBot) handleDeleted(evt *model.WebSocketEvent) {
	post := model.Post{}
	err := json.Unmarshal([]byte(evt.Data["post"].(string)), &post)
	if err != nil {
		log.Error("Could not unmarshall post", err)
		return
	}
	if postApp, ok := mb.postApps[post.Id]; ok {
		log.Info("Unloading app")
		postApp.Stop()
	}
}
