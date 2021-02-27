package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

func (mb *MatterlessBot) postFunctionLog(userID string, functionName string, logMessage string) error {
	user := mb.lookupUser(userID)
	channelName := strings.ToLower(fmt.Sprintf("matterless-logs-%s-%s", user.Username, functionName))
	displayName := fmt.Sprintf("MatterLess: Logs: %s: %s", user.Username, functionName)
	ch, err := mb.ensureLogChannel(mb.team.Id, channelName, displayName)
	if err != nil {
		return err
	}
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
	ch, resp := mb.mmClient.CreateChannel(&model.Channel{
		TeamId:      teamID,
		Type:        model.CHANNEL_PRIVATE,
		DisplayName: displayName,
		Name:        name,
		Header:      "",
		Purpose:     "",
	})
	if resp.StatusCode == 400 {
		ch, resp = mb.mmClient.GetChannelByName(name, teamID, "")
		logAPIResponse(resp, "create log channel")
	}
	return ch, nil
}
