$(function() {
    var data = document.getElementById('result-list');
    var conn = new WebSocket('wss://' + window.location.host + '/api/public/scores/live');
    conn.onclose = function(evt) {
        data.textContent = 'Connection closed';
    };
    conn.onmessage = function(evt) {
        results = JSON.parse(evt.data);
        appendScores(results);
        updateChart(results);
    };
});

function appendScores(scores) {
    Object.keys(scores).map(k => scores[k]).forEach(r => {
        var row = $('#result-list').find('#' + r.name);
        // row.find('.teamnumber').html(r.id);
        row.find('.teamname').html(r.name);
        row.find('.points').html(r.score);
        row.find('.service').html(r.service);
        row.find('.ctf').html(r.ctf);
        // row.find('.other').html(r.other);
    });
}

function updateChart(res) {
    const chart = $('#hc_scoreboard').highcharts();
    build_hc_series(res).forEach(series => {
        chart.get(series.id).setData(series.data, false);
    });
    chart.redraw();
}
