# Environment
```yaml
MM_URL: http://host.docker.internal:8065
MM_TOKEN: xuwwa9z56jgb3b859zmasu1n7w
```
----
# MattermostClient: MyBot
```yaml
url: $MM_URL
token: $MM_TOKEN
events:
  posted: 
  - PingPong
  - Help
```
----
# Function: PingPong
```javascript
async function handle(event) {
    event = cleanEvent(event);
    if(event.event == "posted" && event.post.message === "ping") {
        console.log("Got ping, need to pong.");
        let client = connect();
        let me = await client.getMe();
        await client.addReaction(me.id, event.post.id, "ping_pong");
    }    
}
```

---
# Function: Help
```javascript
//help
async function handle(event) {
    event = cleanEvent(event);
    if(event.event == "posted" && event.post.message === "help") {
        let client = connect();
        let post = event.post;
        await client.createPost({
            channel_id: post.channel_id,
            root_id: post.id,
            parent_id: post.id,
            message: `# This is a help text!
Awesome stuff here.`
        })
    }
}
```
---
# Library
```javascript
import mmConnect from "./mm_client.mjs";

// Connects to mattermost via env settings
function connect() {
    return mmConnect(process.env.MM_URL, process.env.MM_TOKEN);
}

function cleanEvent(event) {
    if(event.event == "posted" || event.event == "post_edted") {
        event.post = JSON.parse(event.data.post);
    }
    return event;
}
```
---
# Function: MyHTTPTest
```javascript
function handle(event) {
    console.log("HTTP event", event);
    return {
        status: 200,
        headers: {
            "X-Zef": "Awesome"
        },
        body: "Hello world!"
    };
}
```

---
# APIGateway: MyHTTP
```yaml
endpoints:
    - path: /test
      methods:
        - GET
        - POST
      function: MyHTTPTest
```