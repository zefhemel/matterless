import http from 'http';

import {handle} from "./function.mjs"

const server = http.createServer((req, res) => {
    let data = '';
    req.on('data', chunk => {
        data += chunk;
    })
    req.on('end', () => {
        try {
            let jsonData = JSON.parse(data);
            Promise.resolve(handle(jsonData)).then(result => {
                res.writeHead(200);
                res.end(JSON.stringify(result || {}));
            }).catch(e => {
                res.writeHead(500);
                res.end(jsonError(e));
            })
        } catch (e) {
            res.writeHead(500);
            res.end(jsonError(e));
        }
    })

    function jsonError(e) {
        return JSON.stringify({
            error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
        })
    }
});
console.error("Starting node.js server...");
server.listen(8080);

