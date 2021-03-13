package eventsource

import (
	"fmt"
	"github.com/mattermost/mattermost-server/model"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
	"os"
)

type SlashCommandSource struct {
	externalURL string
	commandID   string
	adminClient *model.Client4
	apiGateway  *APIGatewaySource
	def         *definition.SlashCommandDef
}

func (s *SlashCommandSource) Start() error {
	team, resp := s.adminClient.GetTeamByName(s.def.TeamName, "")
	if resp.Error != nil {
		return resp.Error
	}
	// First find if an existing already exists, if so we'll patch it
	customCommands, resp := s.adminClient.ListCommands(team.Id, true)
	foundExisting := false
	for _, cmd := range customCommands {
		if cmd.Trigger == s.def.Trigger {
			log.Debug("Found existing command, patching")
			cmd.Method = "P"
			cmd.DisplayName = s.def.Trigger
			cmd.Description = "Auto generated by Matterless"
			cmd.URL = s.externalURL
			_, resp := s.adminClient.UpdateCommand(cmd)
			if resp.Error != nil {
				return resp.Error
			}
			s.commandID = cmd.Id
			log.Info("Done")
			foundExisting = true
		}
	}
	// Then, create it anew
	if !foundExisting {
		log.Debug("Creating new command")
		cmd, resp := s.adminClient.CreateCommand(&model.Command{
			TeamId:       team.Id,
			Trigger:      s.def.Trigger,
			Method:       "P",
			AutoComplete: false,
			DisplayName:  s.def.Trigger,
			Description:  "Auto generated by Matterless",
			URL:          s.externalURL,
		})
		if resp.Error != nil {
			return resp.Error
		}
		s.commandID = cmd.Id
	}

	return s.apiGateway.Start()
}

func (s *SlashCommandSource) Stop() {
	if s.commandID != "" {
		_, resp := s.adminClient.DeleteCommand(s.commandID)
		if resp.Error != nil {
			log.Error("Could not remove command", resp.Error)
		}
	}
	s.apiGateway.Stop()
}

func NewSlashCommandSource(adminClient *model.Client4, def *definition.SlashCommandDef, invokeFunc FunctionInvokeFunc) *SlashCommandSource {
	scs := &SlashCommandSource{
		adminClient: adminClient,
		def:         def,
		apiGateway: NewAPIGatewaySource(&definition.APIGatewayDef{
			BindPort: util.FindFreePort(8200),
			Endpoints: []definition.EndpointDef{{
				Path:     "/callback",
				Methods:  []string{"POST"},
				Function: def.Function,
			}},
		}, invokeFunc),
	}

	// TODO: Let's not pull this from the environment variables here directly
	scs.externalURL = fmt.Sprintf("http://%s:%d/callback", os.Getenv("external_host"), scs.apiGateway.def.BindPort)

	return scs
}

var _ EventSource = &SlashCommandSource{}
