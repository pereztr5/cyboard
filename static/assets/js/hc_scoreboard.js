$(function () {

    var hc_scoreboard_series = {
        name: 'Points',
        id: ':scores',
        data: [],
        dataLabels: {
            enabled: true,
            color: '#FFFFFF',
            align: 'center',
            y: 25, // 25 pixels down from the top
            style: {
                fontSize: '3.2vw'
            }
        }
    };

    // Get initial chart data, set up columns for teams
    $.get( '/team/scores' )
    .done(function(data) {
        $.each(data, function(index, result) {
            hc_scoreboard_series.data.push([
                result.teamname, result.points
            ]);
        });

        $('#hc_scoreboard').highcharts({
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
                labels: {
                    rotation: -45,
                },
                crosshair: false
            },
            yAxis: {
                min: 0,
                title: {
                    text: 'Points'
                }
            },
            legend: {
                enabled: false
            },
            plotOptions: {
                column: {
                    pointPadding: 0,
                    groupPadding: 0.1,
                    borderWidth: 0,
                    colorByPoint: true,
                    shadow: true
                }
            },
            series: [hc_scoreboard_series]
        });
    });

});
