# Cron
```yaml
- schedule: "*/2 * * * *" 
  function: Often
- schedule: "*/3 * * * *"
  function: Often
```

# Function: Often

```javascript
function handle(evt) {
    console.log("YO", evt)
}
```
