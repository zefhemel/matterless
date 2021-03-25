package bot

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/application"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

var safeChannelRegexp *regexp.Regexp = regexp.MustCompile(`[^A-Za-z0-9\-]`)

func safeChannelName(name string) string {
	return strings.ToLower(safeChannelRegexp.ReplaceAllString(name, "-"))
}

func (mb *MatterlessBot) listenForLogs(userID string, app *application.Application) {
	for le := range app.Logs() {
		if le.Instance == nil {
			// Init error, parse errors, don't ship to logger
			continue
		}
		mb.postFunctionLog(userID, le.Instance.Name(), le.Message)
	}
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
