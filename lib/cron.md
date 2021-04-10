# Cron
This Matterless definition file defines a `cron` macro. You can define multiple crons in one go, example use:

    # cron MyCron
    ```
    SomeMeaningfullName:
      schedule: "* * * * * *"
      function: EverySecond
    ```

# macro cron
Implements a simple cronjob scheduler.

```yaml
input_schema:
   type: object
   additionalProperties: 
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

    # Job: CronJob
    ```yaml
    init:
      {{yaml $input | prefixLines "  "}}
    ```
    ```javascript
    import {cron} from 'https://deno.land/x/deno_cron@v1.0.0/cron.ts';
    import {publishEvent} from "./matterless.ts";
    
    function init(config) {
        console.log("Config", config)
        Object.keys(config).forEach(cronName => {
            let cronDef = config[cronName];
            cron(cronDef.schedule, () => {
                publishEvent(`cron:${cronName}`, {});
            });
        });
    }
    ```
    
    # Events
    ```
    {{range $cronName, $def := $input}}
    "cron:{{$cronName}}":
    - {{$def.function}}
    {{- end}}
    ```
