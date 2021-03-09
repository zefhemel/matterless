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
# Source: Me
This is some documentation

```yaml
URL: $MattermostURL
Token: $MattermostToken
```
---
# Subscription: TestSubscription
```yaml
Source: Me
Function: TestFunction1
Events: 
- posted
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