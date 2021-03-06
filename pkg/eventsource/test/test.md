# Source: TestBot1
```yaml
Type: Mattermost
URL: http://localhost:8065
Token: woam7en3obyntdpztapp1o33go
```
# Subscription: TestSubscription
```yaml
Source: TestBot1
Function: TestFunction1
Events: 
- posted
```

------
# Function TestFunction1

```JavaScript
export default function (event) {
  console.log("Got event", event);
}
```