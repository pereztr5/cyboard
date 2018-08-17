$(function() {
    const stat = document.getElementById('status');
    const conn = new WebSocket('wss://' + window.location.host + '/api/public/services/live');
    conn.onclose = (evt) => {
        stat.textContent = 'Connection closed';
    };
    conn.onmessage = (evt) => {
        results = JSON.parse(evt.data);
        appendScores(results)
    };

});

function appendScores(res) {
    const icons = 'fa-arrow-circle-up fa-arrow-circle-down fa-exclamation-circle fa-question-circle-o text-success text-danger text-warning text-muted blink';
    res.forEach((resp) => {
        const group = $('div').find('[data-check="' + resp.service_id + '"]');
        r.statuses.forEach((status, idx) => {
            const stat = group.find('[data-team=' + idx + ']');
            stat.removeClass(icons);

            let newIcon;
            switch(status) {
            case 'pass': newIcon = 'fa-arrow-circle-up text-success'; break;
            case 'fail': newIcon = 'fa-arrow-circle-down text-danger blink'; break;
            case 'partial': newIcon = 'fa-exclamation-circle text-warning'; break;
            default:     newIcon = 'fa-question-circle-o text-muted'; break;
            }
            stat.addClass(newIcon);
        });
    });
}

