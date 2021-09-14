import {serve} from "https://deno.land/std@0.91.0/http/server.ts";
// @ts-ignore
import {init, run, start, stop} from "./function.js"

const port = +Deno.args[0];
const server = serve({hostname: "0.0.0.0", port: port});
console.log(`Starting deno job runtime (${port})`);

// @ts-ignore
Promise.resolve(init()).then(async () => {
    const headers = new Headers();
    headers.set("Content-type", "application/json");
    for await (const request of server) {
        if (request.url === "/start") {
            // @ts-ignore
            Promise.resolve(start()).then(result => {
                request.respond({
                    status: 200,
                    headers: headers,
                    body: JSON.stringify(result || {})
                });
                // Kick off the run() function asynchronously
                // @ts-ignore
                Promise.resolve(run()).catch(e => {
                    console.error(e);
                });
            }).catch(e => {
                request.respond({
                    status: 500,
                    headers: headers,
                    body: jsonError(e)
                });
            });
        } else if (request.url === "/stop") {
            // @ts-ignore
            Promise.resolve(stop()).then(result => {
                request.respond({
                    status: 200,
                    headers: headers,
                    body: JSON.stringify(result || {})
                });
                setTimeout(() => {
                    Deno.exit(0);
                });
            }).catch(e => {
                request.respond({
                    status: 500,
                    headers: headers,
                    body: jsonError(e)
                });
            });
        }

        function jsonError(e: Error) {
            return JSON.stringify({
                error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
            })
        }
    }
});
