# Protocol with child process:
Uses of streams:
* `stdin` will receive JSON objects ending with a newline for new events to trigger an invocation
* `stdout` will be used to send logs (in any format) as a response to an invocation. When the invocation has finished will send `!!EOL!!\n` as a separator
* `stderr` will be used to send a response JSON object.

Invocations will always happen sequentially.