# Bot: SuperBot
```yaml
team_names:
  - Dev
username: super-bot
display_name: Super bot
description: My Materless bot
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
        let client = mmConnect(process.env.SUPERBOT_URL, process.env.SUPERBOT_TOKEN);
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
        let client = mmConnect(process.env.SUPERBOT_URL, process.env.SUPERBOT_TOKEN);
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

function cleanEvent(event) {
    if(event.event == "posted" || event.event == "post_edted") {
        event.post = JSON.parse(event.data.post);
    }
    return event;
}
```
---
# APIGateway: MyHTTP
```yaml
bind_port: 8222
endpoints:
    - path: /
      methods:
        - GET
      function: HTTPIndex
```
---
# Function: HTTPIndex
```javascript
function handle(event) {
    return {
        status: 200,
        body: "Hello world!"
    };
}
```
---
# SlashCommand: ZefCommand
```yaml
team_name: Dev
trigger: zef
function: ZefCommand
```

---
# Function: ZefCommand
```javascript
function handle(event) {
    return {
        status: 200,
        body: {
            text: "You told me this: " + event.form_values.text,
        }
    };
}
```
