const $logViewerPanel = $('.log-viewer-panel')
    , $logViewerForm = $logViewerPanel.find('form')
    , $logDisplay = $logViewerForm.find('textarea');

let logWS;

const FORCE_SCROLL = true;
function followNewLogs(force=false) {
    if (force) {
        $logDisplay.scrollTop($logDisplay[0].scrollHeight);
        return;
    }

    const autoScrollTolerance = 300;

    const bottom = $logDisplay[0].scrollHeight;
    const current = $logDisplay.scrollTop() + $logDisplay.height();

    if ((bottom - current) < autoScrollTolerance) {
        $logDisplay.scrollTop(bottom);
    }
};

$logViewerForm.on('input', '.file_select select', function viewLogFile(e) {
    const $select = $(this);
    const log_file = $select.val();

    if (log_file === "...") { return; }

    const url = `/api/ctf/logs/${log_file}`;
    $.get(url).done(data => {
        $logDisplay.val(data);
        subscribeToLog(log_file);
        followNewLogs(FORCE_SCROLL);
    }).fail(xhr => {
        alert(`Failed to fetch files for ${log_file}: ${getXhrErr(xhr)}`);
    });
});

function subscribeToLog(log_file) {
    // Clean up any previous websocket
    unsubscribeFromLog();

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const endpoint = `${protocol}//${window.location.host}/api/ctf/logs/${log_file}/tail`;
    logWS = new WebSocket(endpoint);

    // Append log data as it comes in.
    logWS.onmessage = (evt) => {
        $logDisplay.val( (_, val) => val + evt.data );
        followNewLogs();
    };
    logWS.onclose = (evt) => {
        $logDisplay.val( (_, val) => val + `\n\n-- WARNING: Live view disconnected!` );
        console.error(evt);
    }
}

function unsubscribeFromLog() {
    if (!logWS) { return; }

    logWS.onclose = null;
    logWS.close();
}
