# Function: LogEvent

```
function handle(event) {
    return {
            text: "Hey there: " + event.form_values.text
    };
}
```
----
# SlashCommand: Zef
```
trigger: zef-test
team_name: Dev
function: LogEvent
```

----
# Bot: SuperZefBot
```
team_names:
- Dev
username: super-zef-bot
```