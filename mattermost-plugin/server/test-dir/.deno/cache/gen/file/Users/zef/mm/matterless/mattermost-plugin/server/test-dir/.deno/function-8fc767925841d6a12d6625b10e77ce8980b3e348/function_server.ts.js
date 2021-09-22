import { serve } from "https://deno.land/std@0.91.0/http/server.ts";
import { handle, init } from "./function.js";
const port = +Deno.args[0];
const server = serve({ hostname: "0.0.0.0", port: port });
console.log(`Starting deno function runtime.`);
const textDecoder = new TextDecoder();
try {
    Promise.resolve(init()).then(async () => {
        for await (const request of server) {
            if (request.method === "POST") {
                const headers = new Headers();
                headers.set("Content-type", "application/json");
                try {
                    const textBody = textDecoder.decode(await Deno.readAll(request.body));
                    const jsonData = JSON.parse(textBody);
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
                    });
                }
                catch (e) {
                    request.respond({
                        status: 500,
                        headers: headers,
                        body: jsonError(e)
                    });
                }
            }
            function jsonError(e) {
                return JSON.stringify({
                    error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
                });
            }
        }
    }).catch(e => {
        console.log("HERE CATCHING 1");
        server.close();
        throw e;
    });
}
catch (e) {
    console.log("HERE CATCHING 2", e);
    console.trace(e);
    Deno.exit(1);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnVuY3Rpb25fc2VydmVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZnVuY3Rpb25fc2VydmVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBQyxLQUFLLEVBQUMsTUFBTSw2Q0FBNkMsQ0FBQztBQUVsRSxPQUFPLEVBQUMsTUFBTSxFQUFFLElBQUksRUFBQyxNQUFNLGVBQWUsQ0FBQTtBQUUxQyxNQUFNLElBQUksR0FBRyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUM7QUFDM0IsTUFBTSxNQUFNLEdBQUcsS0FBSyxDQUFDLEVBQUMsUUFBUSxFQUFFLFNBQVMsRUFBRSxJQUFJLEVBQUUsSUFBSSxFQUFDLENBQUMsQ0FBQztBQUN4RCxPQUFPLENBQUMsR0FBRyxDQUFDLGlDQUFpQyxDQUFDLENBQUM7QUFDL0MsTUFBTSxXQUFXLEdBQUcsSUFBSSxXQUFXLEVBQUUsQ0FBQztBQUV0QyxJQUFJO0lBRUEsT0FBTyxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsQ0FBQyxDQUFDLElBQUksQ0FBQyxLQUFLLElBQUksRUFBRTtRQUNwQyxJQUFJLEtBQUssRUFBRSxNQUFNLE9BQU8sSUFBSSxNQUFNLEVBQUU7WUFDaEMsSUFBSSxPQUFPLENBQUMsTUFBTSxLQUFLLE1BQU0sRUFBRTtnQkFDM0IsTUFBTSxPQUFPLEdBQUcsSUFBSSxPQUFPLEVBQUUsQ0FBQztnQkFDOUIsT0FBTyxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsa0JBQWtCLENBQUMsQ0FBQztnQkFDaEQsSUFBSTtvQkFDQSxNQUFNLFFBQVEsR0FBRyxXQUFXLENBQUMsTUFBTSxDQUFDLE1BQU0sSUFBSSxDQUFDLE9BQU8sQ0FBQyxPQUFPLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztvQkFDdEUsTUFBTSxRQUFRLEdBQUcsSUFBSSxDQUFDLEtBQUssQ0FBQyxRQUFRLENBQUMsQ0FBQztvQkFFdEMsT0FBTyxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUMsUUFBUSxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEVBQUU7d0JBQzVDLE9BQU8sQ0FBQyxPQUFPLENBQUM7NEJBQ1osTUFBTSxFQUFFLEdBQUc7NEJBQ1gsT0FBTyxFQUFFLE9BQU87NEJBQ2hCLElBQUksRUFBRSxJQUFJLENBQUMsU0FBUyxDQUFDLE1BQU0sSUFBSSxFQUFFLENBQUM7eUJBQ3JDLENBQUMsQ0FBQztvQkFDUCxDQUFDLENBQUMsQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLEVBQUU7d0JBQ1QsT0FBTyxDQUFDLE9BQU8sQ0FBQzs0QkFDWixNQUFNLEVBQUUsR0FBRzs0QkFDWCxPQUFPLEVBQUUsT0FBTzs0QkFDaEIsSUFBSSxFQUFFLFNBQVMsQ0FBQyxDQUFDLENBQUM7eUJBQ3JCLENBQUMsQ0FBQztvQkFDUCxDQUFDLENBQUMsQ0FBQTtpQkFDTDtnQkFBQyxPQUFPLENBQUMsRUFBRTtvQkFDUixPQUFPLENBQUMsT0FBTyxDQUFDO3dCQUNaLE1BQU0sRUFBRSxHQUFHO3dCQUNYLE9BQU8sRUFBRSxPQUFPO3dCQUNoQixJQUFJLEVBQUUsU0FBUyxDQUFDLENBQUMsQ0FBQztxQkFDckIsQ0FBQyxDQUFDO2lCQUNOO2FBQ0o7WUFFRCxTQUFTLFNBQVMsQ0FBQyxDQUFRO2dCQUN2QixPQUFPLElBQUksQ0FBQyxTQUFTLENBQUM7b0JBQ2xCLEtBQUssRUFBRSxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxTQUFTLENBQUMsQ0FBQyxFQUFFLE1BQU0sQ0FBQyxtQkFBbUIsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO2lCQUN0RSxDQUFDLENBQUE7WUFDTixDQUFDO1NBQ0o7SUFDTCxDQUFDLENBQUMsQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLEVBQUU7UUFDVCxPQUFPLENBQUMsR0FBRyxDQUFDLGlCQUFpQixDQUFDLENBQUM7UUFDL0IsTUFBTSxDQUFDLEtBQUssRUFBRSxDQUFDO1FBQ2YsTUFBTSxDQUFDLENBQUM7SUFDWixDQUFDLENBQUMsQ0FBQztDQUNOO0FBQUMsT0FBTyxDQUFDLEVBQUU7SUFDUixPQUFPLENBQUMsR0FBRyxDQUFDLGlCQUFpQixFQUFFLENBQUMsQ0FBQyxDQUFDO0lBQ2xDLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLENBQUM7SUFDakIsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUMsQ0FBQztDQUNoQiJ9