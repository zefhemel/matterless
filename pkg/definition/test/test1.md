# Function: TestFunction1

```
function handle(event) {
  console.log("Hello world!");
}
```

------
# Function TestFunction2

```javascript
function handle(event) {
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
function myFunction() {
    
}
```

---
# API
```yaml
- path: /test
  methods:
    - GET
  function: TestFunction2
```

---
# SlashCommand: MyCommand
```yaml
trigger: test
team_name: bla
function: TestFunction1
```

---
# Bot: MyBot
```yaml
username: my-bot
team_names:
  - Bla
events:
  posted:
  - TestFunction1
```

---
# Cron
```yaml
- schedule: 0 * * * * *
  function: MyRepeatedTask
```

---
# Function: MyRepeatedTask
```javascript
function handle() {
    console.log("Triggered");
}
```

# MattermostClient: MyClient
```
url: http://localhost
token: abc
```