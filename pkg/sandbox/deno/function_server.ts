import { serve } from "https://deno.land/std@0.91.0/http/server.ts";
// @ts-ignore
import {init, handle} from "./function.js"

const port = +Deno.args[0];
const server = serve({ hostname: "0.0.0.0", port: port });
console.log(`Starting deno function runtime.`);
const textDecoder = new TextDecoder();

try {
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
                    }).catch(e => {
                        request.respond({
                            status: 500,
                            headers: headers,
                            body: jsonError(e)
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
    }).catch(e => {
        console.log("HERE CATCHING 1");
        server.close();
        throw e;
    });
} catch(e) {
    console.log("HERE CATCHING 2", e);
    console.trace(e);
    Deno.exit(1);
}
