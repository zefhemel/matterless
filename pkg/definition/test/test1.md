# Function: TestFunction1

```
export default function(event) {
  console.log("Hello world!");
}
```

------
# Function TestFunction2

```javascript
export default function (event) {
  console.log("Hello world 2!");
}
```
---
# MattermostClient: Me
This is some documentation

```yaml
url: $MattermostURL
token: $MattermostToken
```

---
# Environment

```yaml
MattermostURL: http://localhost:8065
MattermostToken: 1234
```

---
# Library

```javascript
import connect from "./mm_client.mjs";

// Connects to mattermost via env settings
function connect() {
    return connect(process.env.MM_URL, process.env.MM_TOKEN);
}
```

---
# APIGateway: MyHTTP
```yaml
endpoints:
    - path: /test
      methods:
        - GET
      function: TestFunction2
```

---
# SlashCommand: MyCommand
```yaml
trigger: test
function: TestFunction1
```

---
# Bot: MyBot
```yaml
username: my-bot
events:
  posted:
  - TestFunction1
```