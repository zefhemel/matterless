# Function: LogEvent

```
function handle(event) {
    return {
            text: "Hey there: " + event.text
    };
}
```
----
# SlashCommand: Zef
```
trigger: zef-test
auto_complete: true
auto_complete_desc: Zef's awesome command
auto_complete_hint: your-name
team_name: Dev
function: LogEvent
```
