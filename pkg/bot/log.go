package bot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

var safeChannelRegexp *regexp.Regexp = regexp.MustCompile(`[^A-Za-z0-9\-]`)

func safeChannelName(name string) string {
	return strings.ToLower(safeChannelRegexp.ReplaceAllString(name, "-"))
}

func (mb *MatterlessBot) postFunctionLog(userID string, functionName string, logMessage string) error {
	user := mb.lookupUser(userID)
	channelName := safeChannelName(fmt.Sprintf("matterless-logs-%s-%s", user.Username, functionName))
	displayName := fmt.Sprintf("Matterless: Logs: %s: %s", user.Username, functionName)
	ch, err := mb.ensureLogChannel(mb.team.Id, channelName, displayName)
	if err != nil {
		return err
	}
	apiDelay()
	_, resp := mb.mmClient.AddChannelMember(ch.Id, userID)
	logAPIResponse(resp, "add member")
	_, resp = mb.mmClient.CreatePost(&model.Post{
		ChannelId: ch.Id,
		Message:   fmt.Sprintf("Log:\n```%s```", logMessage),
	})
	logAPIResponse(resp, "create log post")
	return nil
}

func (mb *MatterlessBot) ensureLogChannel(teamID, name, displayName string) (*model.Channel, error) {
	if existingChannel := mb.lookupChannelByName(name, teamID); existingChannel != nil {
		return existingChannel, nil
	}

	ch, resp := mb.mmClient.CreateChannel(&model.Channel{
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
