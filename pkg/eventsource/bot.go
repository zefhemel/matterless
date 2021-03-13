package eventsource

import (
	"fmt"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/definition"
	"strings"
)

type BotSource struct {
	botName            string
	def                *definition.BotDef
	mms                *MatterMostSource
	adminClient        *model.Client4
	functionInvokeFunc FunctionInvokeFunc
	BotUserClient      *model.Client4
}

func NewBotSource(adminClient *model.Client4, botName string, def *definition.BotDef, functionInvokeFunc FunctionInvokeFunc) (*BotSource, error) {
	bs := &BotSource{
		botName:            botName,
		def:                def,
		functionInvokeFunc: functionInvokeFunc,
		adminClient:        adminClient,
	}

	botUser, resp := adminClient.GetUserByUsername(def.Username, "")
	var userID string
	if botUser == nil {
		bot, resp := adminClient.CreateBot(&model.Bot{
			Username:    def.Username,
			DisplayName: def.DisplayName,
			Description: def.Description,
		})
		if resp.Error != nil {
			return nil, errors.Wrap(resp.Error, "create bot")
		}
		userID = bot.UserId
	} else {
		userID = botUser.Id
		_, resp := adminClient.PatchBot(userID, &model.BotPatch{
			DisplayName: &def.DisplayName,
			Description: &def.Description,
		})
		if resp.Error != nil {
			return nil, errors.Wrap(resp.Error, "patch bot")
		}
	}
	tokens, resp := adminClient.GetUserAccessTokensForUser(userID, 0, 100)
	if resp.Error != nil {
		return nil, errors.Wrap(resp.Error, "get access tokens")
	}
	for _, token := range tokens {
		success, resp := adminClient.RevokeUserAccessToken(token.Id)
		if !success {
			return nil, errors.Wrap(resp.Error, "revoke token")
		}
	}

	token, resp := adminClient.CreateUserAccessToken(userID, "Matterless token")
	if resp.Error != nil {
		return nil, errors.Wrap(resp.Error, "create new token")
	}

	for _, teamName := range def.TeamNames {
		team, resp := adminClient.GetTeamByName(teamName, "")
		if resp.Error != nil {
			return nil, errors.Wrap(resp.Error, "team lookup")
		}
		_, resp = adminClient.AddTeamMember(team.Id, userID)
		if resp.Error != nil {
			return nil, errors.Wrap(resp.Error, "add bot to team")
		}
	}

	mms, err := NewMatterMostSource(botName, &definition.MattermostClientDef{
		URL:    adminClient.Url,
		Token:  token.Token,
		Events: def.Events,
	}, functionInvokeFunc)

	if err != nil {
		return nil, err
	}

	bs.mms = mms
	bs.BotUserClient = model.NewAPIv4Client(adminClient.Url)
	bs.BotUserClient.SetOAuthToken(token.Token)

	return bs, err
}

func (ag *BotSource) ExposeEnvironment(env map[string]string) {
	env[fmt.Sprintf("%s_URL", strings.ToUpper(ag.botName))] = ag.adminClient.Url
	env[fmt.Sprintf("%s_TOKEN", strings.ToUpper(ag.botName))] = ag.mms.def.Token
}

func (ag *BotSource) Start() error {
	return ag.mms.Start()
}

func (ag *BotSource) Stop() {

}

var _ EventSource = &BotSource{}
