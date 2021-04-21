# Cron
This Matterless definition file defines a `cron` macro. You can define multiple crons using multiple definitions.

Example use:

    # cron MyCron
    ```
    schedule: "* * * * * *"
    function: EverySecond
    ```

# macro cron
Implements a simple cronjob scheduler.

```yaml
schema:
   type: object
   properties:
      schedule:
        type: string
      function:
        type: string
   required:
   - schedule
   - function
   additionalProperties: false
```

Template:

    # job CronJob
    ```yaml
    init:
      {{$name}}:
        {{yaml $arg | prefixLines "    "}}
    ```

    ```javascript
    import {cron} from 'https://deno.land/x/deno_cron@v1.0.0/cron.ts';
    import {events} from "./matterless.ts";
    
    function init(config) {
        Object.keys(config).forEach(cronName => {
            let cronDef = config[cronName];
            cron(cronDef.schedule, () => {
                events.publish(`cron:${cronName}`, {});
            });
        });
    }
    ```
    
    # events
    ```
    "cron:{{$name}}":
    - {{$arg.function}}
    ```
