const SCOREBOARD_CATEGORIES = ["Service", "CTF", "Other"];

function build_hc_series(scores, categories = SCOREBOARD_CATEGORIES) {
    return categories.map(cat => {
        const cat_l = cat.toLowerCase();
        return {
            id: cat,
            name: cat,
            data: scores.map(row => row[cat_l]),
        };
    });
}

function build_hc_cfg(series, teams) {
    return {
        chart: { type: 'column' },
        title: { text: 'Team Scores' },
        subtitle: { text: '(Updates automatically)' },
        xAxis: {
            type: 'category',
            categories: teams,
            labels: {
                align: 'center',
                style: {
                    fontSize: '14pt',
                    fontWeight: 'bold',
                    textOutline: '1px contrast'
                }
            },
            tickWidth: 0,
            crosshair: false,
        },
        yAxis: {
            min: 0,
            title: {
                text: 'Points'
            },
            stackLabels: {
                enabled: true,
                // Fixes scores above columns from disappearing
                allowOverlap: true,
                // Fixes the 'bad spacing' of scores over 1000 (otherwise rendered as '1 000')
                formatter: function() { return this.total; },
                style: {
                    color: (Highcharts.theme && Highcharts.theme.contrastTextColor) || 'white',
                    fontSize: '3.5vw',
                }
            },
        },
        plotOptions: {
            column: {
                stacking: 'normal',
                pointPadding: 0,
                groupPadding: 0.1,
                // borderWidth: 0,
                // colorByPoint: true,
                shadow: true,
                dataLabels: {
                    enabled: true,
                    formatter: function() { return this.y; },
                    style: {
                        fontSize: '2.2em',
                    },
                },
            }
        },
        // Legend is a floating box on the top-right of the chart
        legend: {
            floating: true,
            verticalAlign: 'top',
            y: 25,
            align: 'right',
            x: -30,
            backgroundColor: (Highcharts.theme && Highcharts.theme.background2) || 'gray',
            borderColor: '#CCC',
            borderWidth: 1,
            shadow: true,
        },
        // Tooltip is the box that appears when hovering over a specific column
        tooltip: {
            shared: true,
            useHTML: true,
            headerFormat: '<span>{point.key}</span><table style="background-color:initial">',
            pointFormat: '<tr>' +
                '<td style="color:{series.color};padding:0">{series.name}: </td>' +
                '<td style="padding:0"><b>{point.y:.0f}</b></td>' +
                '</tr>',
            footerFormat: '</table>',
            hideDelay: 100,
            style: {
                fontSize: '22pt',
            },
        },
        series: series,
    };
}

// Get initial chart data, set up columns for teams
function init_scoreboard() {
    $.getJSON( '/api/public/scores' )
    .done(function(scores) {
        const teams = [...new Set(scores.map(row => row.name))];
        const hc_series = build_hc_series(scores);

        const hc_cfg = build_hc_cfg(hc_series, teams);
        Highcharts.chart('hc_scoreboard', hc_cfg);
    });
}

// Subscribe to the live update endpoint, adjust the table & graph as needed
function init_scoreboard_updater_ws() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const endpoint = `${protocol}//${window.location.host}/api/public/scores/live`;

    const conn = new WebSocket(endpoint);
    conn.onmessage = function(evt) {
        const results = JSON.parse(evt.data);
        sync_score_table(results);
        sync_scoreboard(results);
    };
    conn.onclose = function(evt) {
        const chart = $('#hc_scoreboard').highcharts();
        const warning_subtitle = {
            text: 'Connection closed. Reload to update!',
            style: { color: 'firebrick', fontWeight: 'bold' },
        };
        chart.setSubtitle(warning_subtitle);
    };
}

function sync_score_table(scores) {
    Object.keys(scores).map(k => scores[k]).forEach(r => {
        const row = $('#scoreboard-table').find('#' + r.name);
        // row.find('.teamnumber').html(r.id);
        row.find('.teamname').html(r.name);
        row.find('.points').html(r.score);
        row.find('.service').html(r.service);
        row.find('.ctf').html(r.ctf);
        row.find('.other').html(r.other);
    });
}

function sync_scoreboard(res) {
    const chart = $('#hc_scoreboard').highcharts();
    build_hc_series(res).forEach(series => {
        chart.get(series.id).setData(series.data, false);
    });
    chart.redraw();
}

$(function () {
    init_scoreboard();
    init_scoreboard_updater_ws();
});

