# Function: NoBody
----
# Function: 
----
# Function: GoodFunction

```
function handle() {

}
```

---
# MattermostClient: NoToken

```yaml
url: http://localhost:8065
events:
  posted: 
  - GoodFunction
```
---
# API
```yaml
- function: MyHTTPTest
```
---
# Cron: TestCron
```yaml
- schedule: bla
  function: NonExisting
```
