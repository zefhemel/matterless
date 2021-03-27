import http from 'http';

import {init, start, stop, run} from "./function.mjs"

const server = http.createServer((req, res) => {
    if(req.url === "/start") {
        let data = '';
        req.on('data', chunk => {
            data += chunk;
        })
        req.on('end', () => {
            try {
                let jsonData = JSON.parse(data);
                Promise.resolve(start(jsonData)).then(result => {
                    res.writeHead(200);
                    res.end(JSON.stringify(result || {}));
                    // Kick off the run() function asynchronously
                    run();
                }).catch(e => {
                    res.writeHead(500);
                    res.end(jsonError(e));
                })
            } catch (e) {
                res.writeHead(500);
                res.end(jsonError(e));
            }
        })
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

