module github.com/zefhemel/matterless/mattermost-plugin

go 1.16

require (
	github.com/go-git/go-git/v5 v5.3.0
	github.com/gorilla/websocket v1.4.2
	github.com/mattermost/mattermost-plugin-starter-template/build v0.0.0-20210429201558-f5cae51a20a8
	github.com/mattermost/mattermost-server/v6 v6.0.0-20210921224403-e57531527548
	github.com/nats-io/nats-server/v2 v2.6.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/zefhemel/matterless v0.0.0-04251888ab13090d5304a64972f93e6d8a9f505a
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/zefhemel/matterless => ../
