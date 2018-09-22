$(function() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const endpoint = `${protocol}//${window.location.host}/api/public/services/live`;
    const conn = new WebSocket(endpoint);

    conn.onclose = (evt) => {
        const $statusNode = $('#status');
        $statusNode.textContent = 'Connection closed';
    };
    conn.onmessage = (evt) => {
        results = JSON.parse(evt.data);
        sync_services(results)
    };
});

function sync_services(data) {
    /* data looks like: [{
        "service_id":1,
        "service_name":"WWW Content",
        "statuses": ["partial", "pass", ...],
     },
     {...}, ...]
    */
    data.forEach(service => {
        const $group = $('div').find(`[data-check="${service.service_id}"]`);
        service.statuses.forEach((status, idx) => {
            const $statusBox = $group.find(`[data-team=${idx}]`);

            // If the status hasn't changed, skip past this box/node.
            if($statusBox.attr('data-status') === status) {
                return;
            }

            let newIcon;
            switch(status) {
            case 'pass': newIcon = 'fa-arrow-circle-up text-success'; break;
            case 'fail': newIcon = 'fa-arrow-circle-down text-danger blink'; break;
            case 'partial': newIcon = 'fa-exclamation-circle text-warning'; break;
            default:     newIcon = 'fa-question-circle text-muted'; break;
            }
            $statusBox.attr('class', `fa ${newIcon}`)
                      .attr('data-status', status);
        });
    });
}

