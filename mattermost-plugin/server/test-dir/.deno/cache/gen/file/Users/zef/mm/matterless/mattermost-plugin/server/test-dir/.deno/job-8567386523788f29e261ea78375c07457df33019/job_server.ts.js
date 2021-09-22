import { serve } from "https://deno.land/std@0.91.0/http/server.ts";
import { init, run, start, stop } from "./function.js";
const port = +Deno.args[0];
const server = serve({ hostname: "0.0.0.0", port: port });
console.log(`Starting deno job runtime (${port})`);
Promise.resolve(init()).then(async () => {
    const headers = new Headers();
    headers.set("Content-type", "application/json");
    for await (const request of server) {
        if (request.url === "/start") {
            Promise.resolve(start()).then(result => {
                request.respond({
                    status: 200,
                    headers: headers,
                    body: JSON.stringify(result || {})
                });
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
        }
        else if (request.url === "/stop") {
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
        function jsonError(e) {
            return JSON.stringify({
                error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
            });
        }
    }
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiam9iX3NlcnZlci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImpvYl9zZXJ2ZXIudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFDLEtBQUssRUFBQyxNQUFNLDZDQUE2QyxDQUFDO0FBRWxFLE9BQU8sRUFBQyxJQUFJLEVBQUUsR0FBRyxFQUFFLEtBQUssRUFBRSxJQUFJLEVBQUMsTUFBTSxlQUFlLENBQUE7QUFFcEQsTUFBTSxJQUFJLEdBQUcsQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQyxDQUFDO0FBQzNCLE1BQU0sTUFBTSxHQUFHLEtBQUssQ0FBQyxFQUFDLFFBQVEsRUFBRSxTQUFTLEVBQUUsSUFBSSxFQUFFLElBQUksRUFBQyxDQUFDLENBQUM7QUFDeEQsT0FBTyxDQUFDLEdBQUcsQ0FBQyw4QkFBOEIsSUFBSSxHQUFHLENBQUMsQ0FBQztBQUduRCxPQUFPLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxDQUFDLENBQUMsSUFBSSxDQUFDLEtBQUssSUFBSSxFQUFFO0lBQ3BDLE1BQU0sT0FBTyxHQUFHLElBQUksT0FBTyxFQUFFLENBQUM7SUFDOUIsT0FBTyxDQUFDLEdBQUcsQ0FBQyxjQUFjLEVBQUUsa0JBQWtCLENBQUMsQ0FBQztJQUNoRCxJQUFJLEtBQUssRUFBRSxNQUFNLE9BQU8sSUFBSSxNQUFNLEVBQUU7UUFDaEMsSUFBSSxPQUFPLENBQUMsR0FBRyxLQUFLLFFBQVEsRUFBRTtZQUUxQixPQUFPLENBQUMsT0FBTyxDQUFDLEtBQUssRUFBRSxDQUFDLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxFQUFFO2dCQUNuQyxPQUFPLENBQUMsT0FBTyxDQUFDO29CQUNaLE1BQU0sRUFBRSxHQUFHO29CQUNYLE9BQU8sRUFBRSxPQUFPO29CQUNoQixJQUFJLEVBQUUsSUFBSSxDQUFDLFNBQVMsQ0FBQyxNQUFNLElBQUksRUFBRSxDQUFDO2lCQUNyQyxDQUFDLENBQUM7Z0JBR0gsT0FBTyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUUsQ0FBQyxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsRUFBRTtvQkFDN0IsT0FBTyxDQUFDLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQztnQkFDckIsQ0FBQyxDQUFDLENBQUM7WUFDUCxDQUFDLENBQUMsQ0FBQyxLQUFLLENBQUMsQ0FBQyxDQUFDLEVBQUU7Z0JBQ1QsT0FBTyxDQUFDLE9BQU8sQ0FBQztvQkFDWixNQUFNLEVBQUUsR0FBRztvQkFDWCxPQUFPLEVBQUUsT0FBTztvQkFDaEIsSUFBSSxFQUFFLFNBQVMsQ0FBQyxDQUFDLENBQUM7aUJBQ3JCLENBQUMsQ0FBQztZQUNQLENBQUMsQ0FBQyxDQUFDO1NBQ047YUFBTSxJQUFJLE9BQU8sQ0FBQyxHQUFHLEtBQUssT0FBTyxFQUFFO1lBRWhDLE9BQU8sQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLENBQUMsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEVBQUU7Z0JBQ2xDLE9BQU8sQ0FBQyxPQUFPLENBQUM7b0JBQ1osTUFBTSxFQUFFLEdBQUc7b0JBQ1gsT0FBTyxFQUFFLE9BQU87b0JBQ2hCLElBQUksRUFBRSxJQUFJLENBQUMsU0FBUyxDQUFDLE1BQU0sSUFBSSxFQUFFLENBQUM7aUJBQ3JDLENBQUMsQ0FBQztnQkFDSCxVQUFVLENBQUMsR0FBRyxFQUFFO29CQUNaLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDLENBQUM7Z0JBQ2pCLENBQUMsQ0FBQyxDQUFDO1lBQ1AsQ0FBQyxDQUFDLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxFQUFFO2dCQUNULE9BQU8sQ0FBQyxPQUFPLENBQUM7b0JBQ1osTUFBTSxFQUFFLEdBQUc7b0JBQ1gsT0FBTyxFQUFFLE9BQU87b0JBQ2hCLElBQUksRUFBRSxTQUFTLENBQUMsQ0FBQyxDQUFDO2lCQUNyQixDQUFDLENBQUM7WUFDUCxDQUFDLENBQUMsQ0FBQztTQUNOO1FBRUQsU0FBUyxTQUFTLENBQUMsQ0FBUTtZQUN2QixPQUFPLElBQUksQ0FBQyxTQUFTLENBQUM7Z0JBQ2xCLEtBQUssRUFBRSxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxTQUFTLENBQUMsQ0FBQyxFQUFFLE1BQU0sQ0FBQyxtQkFBbUIsQ0FBQyxDQUFDLENBQUMsQ0FBQyxDQUFDO2FBQ3RFLENBQUMsQ0FBQTtRQUNOLENBQUM7S0FDSjtBQUNMLENBQUMsQ0FBQyxDQUFDIn0=