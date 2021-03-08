# Environment
```yaml
MM_URL: http://host.docker.internal:8065
MM_TOKEN: xuwwa9z56jgb3b859zmasu1n7w
```
----
# Source: TestBot
```yaml
Type: Mattermost
URL: http://localhost:8065
Token: xuwwa9z56jgb3b859zmasu1n7w
```
----
# Function: PingPong
```javascript
import connect from "./mm_client.mjs";

async function handle(event) {
    if(event.event == "posted") {
        let post = JSON.parse(event.data.post);
        if(post.message === "ping") {
            console.log("Got ping, need to pong.");
            let client = connect(process.env.MM_URL, process.env.MM_TOKEN);
            let me = await client.getMe();
            await client.addReaction(me.id, post.id, "ping_pong");
        }
    }    
}
```
---
# Subscription: TestSubscription
```yaml
Source: TestBot
Function: PingPong
Events:
- posted
- post_edited
```
----
