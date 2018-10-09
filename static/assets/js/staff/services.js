/* Primitive string parser for converting:
    `--attempts 2 -f "/home/costas/Monty Python.mp4" {IP}`
into an array of strings:
    ["--atempts", "2", "-f", "/home/costas/Monty Python.mp4", "{IP}"]

CAVEATS: This only handles one level of quotes, and no escaping rules.
TODO: Should replace with something less fragile (maybe parse server-side, instead?) */
function parseArgs(str) {
    const args = [];
    let inQuotes = false;
    let c = '';
    for(let i = 0; i < str.length; i++) {
        if(str.charAt(i) === ' ' && !inQuotes) {
            args.push(c);
            c = '';
        } else {
            if(str.charAt(i) === '\"') {
                inQuotes = !inQuotes;
            } else {
                c += str.charAt(i);
            }
        }
    }
    if(inQuotes) {
        throw new Error("Args had dangling quotes!");
    }
    args.push(c);
    return args;
};

