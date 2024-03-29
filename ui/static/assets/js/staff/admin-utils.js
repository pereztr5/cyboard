/* Helper methods */

// Send JSON to the server
function ajaxJSON(method, url, data) {
    return $.ajax({
        url,
        method,
        data: JSON.stringify(data),
        contentType: "application/json; charset=utf-8",
    });
}

const getXhrErr = xhr => xhr.status === 0 ? "Network error!" : xhr.responseText;

// Send JSON via POST/PUT to the API, prompt with the response, and on success reload the page.
function ajaxAndReload(method, url, data, successMsg) {
    return ajaxJSON(method, url, data).then(() => {
        alert(successMsg + " Page will reload.");
        window.location.reload();
    }).catch((xhr) => {
        alert(getXhrErr(xhr));
    });
}

// Humanize byte sizes
/* Credits to Faust on Stackoverflow: https://stackoverflow.com/a/39906526 */
function niceBytes(x){
    const units = ['bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    let l = 0, n = parseInt(x, 10) || 0;
    while(n >= 1024 && ++l)
          n = n/1024;
    return(n.toFixed(n >= 10 || l < 1 ? 1 : 2) + ' ' + units[l]);
}

// During a long request, disable a submit button and give it a spinner icon.
// First call creates a function on the associated btn node, which
// can then be called with true/false to enable/disable the loading effect.
function toggleLoadingButton($btn) {
    return state => {
        $btn.prop('disabled', state)
            .find('i').toggleClass('fa-spin fa-spinner', state);
    };
}

// Primitive string parser for converting:
//     `--attempts 2 -f "/home/costas/Monty Python.mp4" {IP}`
// into an array of strings:
//     ["--atempts", "2", "-f", "/home/costas/Monty Python.mp4", "{IP}"]
//
// CAVEATS: This only handles one level of quotes, and no escaping rules.
// TODO: Should replace with something less fragile (maybe parse server-side, instead?)
function splitArgs(str) {
    const args = [];
    let inQuotes = false;
    let arg = '';
    for(let i = 0; i < str.length; i++) {
        // Split on spaces, unless within quotes
        if(str.charAt(i) === ' ' && !inQuotes) {
            args.push(arg);
            arg = ''; // Reset
        } else {
            if(str.charAt(i) === '\"') {
                inQuotes = !inQuotes;
            } else {
                arg += str.charAt(i);
            }
        }
    }
    if(inQuotes) {
        throw new Error("Args had dangling quotes!");
    }
    args.push(arg);
    return args;
};

