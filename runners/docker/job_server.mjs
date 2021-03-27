import http from 'http';

import {init, start, stop, run} from "./function.mjs"

const server = http.createServer((req, res) => {
    if(req.url === "/start") {
        Promise.resolve(start()).then(result => {
            res.writeHead(200);
            res.end(JSON.stringify(result || {}));
            // Kick off the run() function asynchronously
            Promise.resolve(run()).catch(e => {
                console.error(e);
            });
        }).catch(e => {
            res.writeHead(500);
            res.end(jsonError(e));
        });
    } else if(req.url === "/stop") {
        Promise.resolve(stop()).then(result => {
            res.writeHead(200);
            res.end(JSON.stringify(result || {}));
        }).catch(e => {
            res.writeHead(500);
            res.end(jsonError(e));
        });
    }

    function jsonError(e) {
        return JSON.stringify({
            error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
        })
    }
});

Promise.resolve(init()).then(() => {
    server.listen(8080);
});

