$(function() {
    var stat = document.getElementById('status');
    var conn = new WebSocket('wss://' + window.location.host + '/team/services/live');
    conn.onclose = function(evt) {
        stat.textContent = 'Connection closed';
    };
    conn.onmessage = function(evt) {
        results = JSON.parse(evt.data);
        appendScores(results)
    };

});

function appendScores(res) {
    var icons = 'fa-arrow-circle-up fa-arrow-circle-down fa-exclamation-circle fa-question-circle-o text-success text-danger text-warning text-muted blink';
    res.forEach(function(r) {
        var group = $('div').find('[data-check="' + r.service + '"]');
        r.teams.forEach(function(team) {
            var stat = group.find('[data-team=' + team.number + ']');
            var newIcon = 'fa-question-circle-o text-muted';
            stat.removeClass(icons);
            if (team.status === "Status: 0") {
                newIcon = 'fa-arrow-circle-up text-success';
            } else if (team.status === "Status: 2") {
                newIcon = 'fa-arrow-circle-down text-danger blink';
            } else if (team.status === "Status: 1") {
                newIcon = 'fa-exclamation-circle text-warning';
            }
            stat.addClass(newIcon);
        });
    });
}

