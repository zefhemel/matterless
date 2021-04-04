# Import
* file:lib/mattermost.md

Or URL: https://raw.githubusercontent.com/zefhemel/matterless/reset/lib/mattermost.md

# MattermostClient: MyClient
```yaml
url: ${config:url}
token: ${config:token}
events:
  hello:
    - MyCustomFunc
  posted:
    - MyCustomFunc
```

# Function: MyCustomFunc
```javascript

function handle(event) {
    console.log("Got custom event", event);
}
```
