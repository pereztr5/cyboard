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

function joinResultTypes(results) {
    return results.reduce((scores, r) => {
        const score = scores[r.teamname];
        if(score) {
            score[r.type.toLowerCase()] = r.points;
            score.points = score.ctf + score.service;
        } else {
            scores[r.teamname] = {
                teamname: r.teamname,
                teamnumber: r.teamnumber,
                [r.type.toLowerCase()]: r.points,
            };
        }
        return scores;
    }, {})
}

function appendScores(res) {
    const scores = joinResultTypes(res);

    Object.keys(scores).map(k => scores[k]).forEach(r => {
        var row = $('#result-list').find('#' + r.teamname);
        row.find('.teamnumber').html(r.teamnumber);
        row.find('.teamname').html(r.teamname);
        row.find('.points').html(r.points);
        row.find('.service').html(r.service);
        row.find('.ctf').html(r.ctf);
    });
}

function updateChart(res) {
    const chart = $('#hc_scoreboard').highcharts();
    build_hc_series(res).forEach(series => {
        chart.get(series.id).setData(series.data, false);
    });
    chart.redraw();
}
