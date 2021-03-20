# Simple Task manager
Key design: `task:user_id:timestamp`

# SlashCommand: TodoCommand
```yaml
trigger: zodo
auto_complete: true
auto_complete_desc: The ultimate todo command
auto_complete_hint: "[list|add]"
team_name: Dev
function: TodoFunc
```

# Function: TodoFunc
```javascript
import {Store} from "matterless";

let store = new Store();

async function handle(event) {
    if(isWarmupEvent(event)) return;
    if(event.text.startsWith("add ")) {
        let taskTitle = event.text.substring("add ".length).trim();
        let taskId = Date.now();
        await store.put(`task:${event.user_id}:${taskId}`, {
            task_id: taskId,
            user_id: event.user_id,
            title: taskTitle,
            date_created: Date.now(),
            completed: false
        });
        return {
            text: `Added task '${taskTitle}' successfully!`
        }
    } else if(event.text.startsWith("list")) {
        let allTasks = (await store.queryPrefix(`task:${event.user_id}:`));
        if(!allTasks) {
            return {
                text: "No tasks on your list!"
            }
        }
        return {
            attachments: allTasks.map(([key, task]) => {
                return {
                    text: `### ${task.title}
Created _${new Date(task.date_created)}_`,
                    actions: [
                        {
                            id: "complete",
                            name: "Complete",
                            integration: {
                                url: `${process.env.API_URL}/button_callback`,
                                context: {
                                    action: "complete",
                                    task_id: task.task_id
                                }
                            }
                        },
                        {
                            id: "delete",
                            name: "Delete",
                            integration: {
                                url: `${process.env.API_URL}/button_callback`,
                                context: {
                                    action: "delete",
                                    task_id: task.task_id
                                }
                            }
                        }
                    ]
                };
            })
        };
    } else if(event.text.startsWith("clear")) {
        let allTasks = await store.queryPrefix(`task:${event.user_id}:`);
        for(let [key, task] of allTasks) {
            await store.del(key);
        }
        return {
            text: "All cleared."
        }
    }
}
```

# API
```yaml
- path: /button_callback
  function: ButtonCallback
  methods:
    - POST
```

# Function: ButtonCallback
```javascript
import {Store} from "matterless";

let store = new Store();

async function handle(event) {
    if(isWarmupEvent(event)) return;
    let context = event.json_body.context;
    if(context.action === "complete") {
        await store.del(`task:${event.json_body.user_id}:${context.task_id}`);
        return {
            status: 200,
            headers: {
                "Content-type": "application/json"
            },
            body: {
                update: {
                    "message": "Completed!",
                    "props": {}
                }
            }
        };
    }
}
```
