$(function() {
    var stat = document.getElementById('status');
    var conn = new WebSocket('wss://' + window.location.host + '/team/services/live');
    conn.onclose = function(evt) {
        stat.textContent = 'Connection closed';
    }
    conn.onmessage = function(evt) {
        results = JSON.parse(evt.data);
        appendScores(results)
    }
function appendScores(res) {
    var icons = 'fa-check-circle-o fa-times-circle-o fa-exclamation-circle fa-question-circle-o'
    res.forEach(function(r) {
        var group = $('div').find('[data-check="' + r._id + '"]');
        r.teams.forEach(function(team) {
            var stat = group.find('[data-team=' + team.number + ']');
            var newIcon = 'fa-question-circle-o';
            stat.removeClass(icons);
            if (team.status == "Status: 0") {
                newIcon = 'fa-check-circle-o';
            } else if (team.status == "Status: 1") {
                newIcon = 'fa-times-circle-o';
            } else if (team.status == "Status: 2") {
                newIcon = 'fa-exclamation-circle';
            }
            stat.addClass(newIcon);
        });
    });
}

});

