$(function() {
    var data = document.getElementById('result-list');
    var conn = new WebSocket('wss://' + window.location.host + '/team/scores/live');
    conn.onclose = function(evt) {
        data.textContent = 'Connection closed';
    };
    conn.onmessage = function(evt) {
        results = JSON.parse(evt.data);
        appendScores(results);
        updateChart(results);
    };
});

function appendScores(res) {
    res.forEach(function(r) {
        var row = $('#result-list').find('#' + r.teamname);
        row.find('.teamnumber').html(r.teamnumber);
        row.find('.teamname').html(r.teamname);
        row.find('.points').html(r.points);
    });
}

function updateChart(res) {
    $('#hc_scoreboard').highcharts()
        .get(':scores')
        .setData(
            // "scores" is an array of [{teamname}, {score}]
            res.map( function(r) {
                return [r.teamname, r.points];
            })
        );
}
