const SCOREBOARD_CATEGORIES = ["Service", "CTF"];

function build_hc_series(scores, categories = SCOREBOARD_CATEGORIES) {
    return categories.map(cat => ({
        id: cat,
        name: cat,
        data: scores.filter(doc => doc.type === cat).map(doc => doc.points),
    }));
}

$(function () {

    // Get initial chart data, set up columns for teams
    $.getJSON( '/api/public/scores/split' )
    .done(function(scores) {
        const teams = [...new Set(scores.map(doc => doc.teamname))];

        const hc_scoreboard_series = build_hc_series(scores);

        Highcharts.chart('hc_scoreboard', {
            chart: {
                type: 'column'
            },
            title: {
                text: 'Team Scores'
            },
            subtitle: {
                text: '(Updates automatically)'
            },
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
            series: hc_scoreboard_series,
        });
    });

});
