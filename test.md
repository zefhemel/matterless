# import
* file:lib/mattermost.md

# mattermostBot AwesomeBot2
```yaml
url: https://zef.cloud.mattermost.com
admin_token: ${config:admin_token}
username: awesome-bot2
teams:
  - main
events:
  posted: 
    - DebugEvent
```

# function DebugEvent
```javascript
function handle(event) {
    console.log("Got mattermost event", event);
}
```