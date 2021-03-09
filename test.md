# Environment
```yaml
MM_URL: http://host.docker.internal:8065
MM_TOKEN: xuwwa9z56jgb3b859zmasu1n7w
```
----
# Source: TestBot
```yaml
Type: Mattermost
URL: $MM_URL
Token: $MM_TOKEN
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
----
# Function: Help
```javascript
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
# Subscription: PingSubscription
```yaml
Source: TestBot
Events:
- posted
Function: PingPong
```
---
# Subscription: HelpSubscription
```yaml
Source: TestBot
Events:
- posted
Function: Help
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