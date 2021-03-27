package bot

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

var safeChannelRegexp *regexp.Regexp = regexp.MustCompile(`[^A-Za-z0-9\-]`)

func safeChannelName(name string) string {
	return strings.ToLower(safeChannelRegexp.ReplaceAllString(name, "-"))
}

func (mb *MatterlessBot) listenForLogs() {
	mb.eventBus.Subscribe("logs:*", func(eventName string, eventData interface{}) (interface{}, error) {
		if le, ok := eventData.(application.LogEntry); ok {
			if le.LogEntry.Instance == nil {
				return nil, nil
			}
			log.Infof("[App: %s | Function: %s] %s", le.AppName, le.LogEntry.Instance.Name(), le.LogEntry.Message)
			parts := strings.Split(le.AppName, ":")
			if len(parts) != 2 {
				// This is not coming from a messenger bot
				return nil, nil
			}
			mb.postFunctionLog(parts[0], le.LogEntry.Instance.Name(), le.LogEntry.Message)
		} else {
			log.Error("Received log event that's not an application.LogEntry ", eventData)
		}
		return nil, nil
	})
}

func (mb *MatterlessBot) postFunctionLog(userID string, functionName string, logMessage string) error {
	client := mb.botSource.BotUserClient
	user := mb.lookupUser(userID)
	channelName := safeChannelName(fmt.Sprintf("matterless-logs-%s-%s", user.Username, functionName))
	displayName := fmt.Sprintf("Matterless: Logs: %s: %s", user.Username, functionName)
	ch, err := mb.ensureLogChannel(mb.team.Id, channelName, displayName)
	if err != nil {
		return err
	}
	apiDelay()
	_, resp := client.AddChannelMember(ch.Id, userID)
	logAPIResponse(resp, "add member")

	_, resp = client.CreatePost(&model.Post{
		ChannelId: ch.Id,
		Message:   fmt.Sprintf("```\n%s```", logMessage),
	})
	logAPIResponse(resp, "create log post")
	return nil
}

func (mb *MatterlessBot) ensureLogChannel(teamID, name, displayName string) (*model.Channel, error) {
	if existingChannel := mb.lookupChannelByName(name, teamID); existingChannel != nil {
		return existingChannel, nil
	}

	ch, resp := mb.botSource.BotUserClient.CreateChannel(&model.Channel{
		TeamId:      teamID,
		Type:        model.CHANNEL_PRIVATE,
		DisplayName: displayName,
		Name:        name,
		Header:      "",
		Purpose:     "",
	})
	if ch == nil {
		// Error
		return nil, resp.Error
	}
	return ch, nil
}
