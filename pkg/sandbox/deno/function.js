function init() {
    console.log("Initing");
}

function handle(event) {
    console.log("Got event", event);
    return {
        status: "ok"
    }
}
function start() {
    console.log("Starting");
}

function stop() {
    console.log("Stopping");
}

var _init, _start, _run, _stop, _handle;
(function() {
    var config = {};

    // Initialization
    try {
        _init = init.bind(null, config);
    } catch (e) {
        _init = () => {
        };
    }

// Functions
    try {
        _handle = handle;
    } catch (e) {
        _handle = () => {
        };
    }

// Jobs
    try {
        _start = start;
    } catch (e) {
        _start = () => {
        };
    }
    try {
        _run = run;
    } catch (e) {
        _run = () => {
        };
    }

    try {
        _stop = stop;
    } catch (e) {
        _stop = () => {
        };
    }
})();


export {
    _init as init,
    _start as start,
    _stop as stop,
    _run as run,
    _handle as handle
};
