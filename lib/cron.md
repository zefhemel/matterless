# Macro: Cron
```yaml
schema:
  type: object
  properties:
     items: 
       type: array
       items:
          type: object
          properties:
            schedule:
              type: string
            function:
              type: string
          additionalProperties: false
            required:
            - url
            - token

```

# Job: MyCron
```yaml
init:
  items:
    - schedule: "*/10 * * * * *"
      function: MyCronFunction
```
```javascript
import {cron} from 'https://deno.land/x/deno_cron@v1.0.0/cron.ts';
import {publishEvent} from "./matterless.ts";

function init(config) {
    config.items.forEach((entry, i) => {
        cron(entry.schedule, () => {
            publishEvent(`MyCron:schedule-${i}`, {});
        });
    })
}
```

# Events
```
"MyCron:schedule-0":
- MyCronFunction
```

# Function: MyCronFunction
```javascript
function handle() {
    console.log("Triggered!");
}
```
