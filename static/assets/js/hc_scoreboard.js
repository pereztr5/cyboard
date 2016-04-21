$(function () {

    var hc_scoreboard_series = {
        name: "Points",
        id: ":scores",
        data: [],
        // data: [
        //   ["test", 500],
        //   ["local idiot", 100],
        //   ["daniel", 777],
        //   ["brian", 240],
        //   ["james", 690],
        //   ["mike", 675],
        //   ["brody", 480],
        // ],
        dataLabels: {
            enabled: true,
            color: '#FFFFFF',
            align: 'center',
            y: 10, // 10 pixels down from the top
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
                    pointPadding: 0.2,
                    borderWidth: 0,
                    colorByPoint: true
                }
            },
            series: [hc_scoreboard_series]
        });
    });

});
