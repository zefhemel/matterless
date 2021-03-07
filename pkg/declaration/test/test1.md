# Function: TestFunction1

```
export default function(event) {
  console.log("Hello world!");
}
```

------
# Function TestFunction2

```JavaScript
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