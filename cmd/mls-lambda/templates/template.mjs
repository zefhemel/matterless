{{.Code}}

import readline from "readline";

let rl = readline.createInterface({
    input: process.stdin,
    terminal: false
});

rl.on('line', function(line){
    let sentEOL = false;
    try {
        let event = JSON.parse(line);
        let result = handle(event);

        console.log("!!EOL!!");
        sentEOL = true;

        if (result === undefined) {
            console.error(JSON.stringify({}));
        } else if (result.then) {
            result.then((result) => {
                if (result === undefined) {
                    console.error(JSON.stringify({}));
                } else {
                    console.error(JSON.stringify(result));
                }
            }).catch(handleFailure);
        } else {
            console.error(JSON.stringify(result));
        }
    } catch(e) {
        handleFailure(e);
    }

    function handleFailure(e) {
        console.error(JSON.stringify({
            status: "error",
            error: JSON.parse(JSON.stringify(e, Object.getOwnPropertyNames(e)))
        }));
        if(!sentEOL) {
            console.log("!!EOL!!");
        }
    }
});
