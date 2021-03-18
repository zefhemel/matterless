# Basic Example using everything
## Function: TestFunction
```
function handle(event) {
  console.log("Hello world!");
}
```
---
# MattermostClient: Me
```yaml
url: http://localhost:8065
token: $my_token
events:
   all:
     - TestFunction
```

---
# Environment
```yaml
my_token: super-secret
```

---
# Library
```javascript
function whoAmI() {
    return "you";
}
```
---
# APIGateway: MyHTTP
```yaml
endpoints:
    - path: /test
      methods:
        - GET
      function: TestFunction
```
---
# SlashCommand: MyCommand
```yaml
team_name: Dev
trigger: test
function: TestFunction
```

---
# Bot: MyBot
```yaml
username: my-bot
events:
  posted:
  - TestFunction
```

---
# Cron: EveryMinute
```yaml
schedule: 0 * * * * *
function: TestFunction
```
