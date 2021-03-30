import { serve } from "https://deno.land/std@0.91.0/http/server.ts";
// @ts-ignore
import {init, handle} from "./function.js"

const port = +Deno.args[0];
const server = serve({ hostname: "0.0.0.0", port: port });
console.log(`Server listening on: http://localhost:${port}`);
const textDecoder = new TextDecoder();

// @ts-ignore
Promise.resolve(init()).then(async () => {
    for await (const request of server) {
        if(request.method === "POST") {
            const headers = new Headers();
            headers.set("Content-type", "application/json");
            try {
                const textBody = textDecoder.decode(await Deno.readAll(request.body));
                const jsonData = JSON.parse(textBody);
                // @ts-ignore
                Promise.resolve(handle(jsonData)).then(result => {
                    request.respond({
                        status: 200,
                        headers: headers,
                        body: JSON.stringify(result || {})
                    });
                })
            } catch(e) {
                request.respond({
                    status: 500,
                    headers: headers,
                    body: jsonError(e)
                });
            }
        }

        function jsonError(e : Error) {
            return JSON.stringify({
                error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
            })
        }
    }
});
