package definition

import (
	"regexp"

	"github.com/mattermost/mattermost-server/model"
	"gopkg.in/yaml.v2"
)

var multiDocSeparatorRegex = regexp.MustCompile(`\-{3,}`)
var functionDefRegex = regexp.MustCompile("\\s*(define|#)\\s+(\\w+)\\:?\\s+```([A-Za-z]+)?([\\s\\S]+)```\\s*")
var identityDefRegex = regexp.MustCompile("\\s*identity\\s+(\\w+):\\s+(\\w+)")
var subscribeDefRegex = regexp.MustCompile("\\s*[Ss]ubscribe\\s+(\\w+)\\:?\\s+([\\s\\S]+)")

type yamlSubscription struct {
	Identity string   `yaml:"Identity"`
	Channel  string   `yaml:"Channel"`
	Events   []string `yaml:"Events"`
}

func Parse(posts []*model.Post) (Definitions, error) {
	// functionDefs := make([]FunctionDef, 0, 5)
	definitions := Definitions{
		Functions:     map[string]FunctionDef{},
		Identities:    map[string]IdentityDef{},
		Subscriptions: []SubscriptionDef{},
	}
	for _, p := range posts {
		blocks := multiDocSeparatorRegex.Split(p.Message, -1)
		for _, block := range blocks {
			// Check if this is a FunctionDef
			defMatchParts := functionDefRegex.FindStringSubmatch(block)
			if defMatchParts != nil {
				// fmt.Printf("Name: %s Language: %s Code: %s", defMatchParts[2], defMatchParts[3], defMatchParts[4])
				definitions.Functions[defMatchParts[2]] = FunctionDef{
					Language: defMatchParts[3],
					Name:     defMatchParts[2],
					Code:     defMatchParts[4],
				}
				continue
			}
			defMatchParts = identityDefRegex.FindStringSubmatch(block)
			if defMatchParts != nil {
				definitions.Identities[defMatchParts[1]] = IdentityDef{
					Token: defMatchParts[2],
				}
				continue
			}
			defMatchParts = subscribeDefRegex.FindStringSubmatch(block)
			if defMatchParts != nil {
				yamlBody := defMatchParts[2]
				var yamlSub yamlSubscription
				err := yaml.Unmarshal([]byte(yamlBody), &yamlSub)
				if err != nil {
					return definitions, err
				}
				definitions.Subscriptions = append(definitions.Subscriptions, SubscriptionDef{
					TriggerFunction: defMatchParts[1],
					EventTypes:      yamlSub.Events,
					Channel:         yamlSub.Channel,
					Identity:        yamlSub.Identity,
				})
				continue
			}
		}
	}
	return definitions, nil
}
