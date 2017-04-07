(function() {
    var user_cfg = $('#user-config-table');
    user_cfg.editableTableWidget({});

    var placeholderUsersCsv =
        "     Name,     Group, Number,            IP, Password\n" +
        "    team1,  blueteam,      1,  192.168.90.1, smokestack\n" +
        "    team2,  blueteam,      2,  192.168.90.2, bologna\n" +
        "    team3,  blueteam,      3,  192.168.90.3, xmas_monkEE\n" +
        "    team4,  blueteam,      4,  192.168.90.4, d2hhdCB0aGUgZnVjayBkaWQgeW91IGV4cGVjdAo=\n" +
        " evilcorp,   redteam,    100, 192.168.199.1, \"rUsstED\"\"CuR^mug3Ons\"\n" +
        "   himike, whiteteam,    255,       1.1.1.1, challenge--m4ster\n" +
        " johnWifi, whiteteam,    256,       0.0.0.0, whatsAfirewall?\n" +
        "lugerJose, blackteam,    666,     6.6.6.255, \"!*,B4NGb4ng,*!\"\n";

    $('.user-config-csv-upload').find('textarea')
        .val(placeholderUsersCsv);

    $('.user-config-csv-upload form').on('submit', function(e) {
        e.preventDefault();
        submitTeamsTextareaCsv();
    })
})();

var submitTeamsTextareaCsv = function() {
    $.ajax({
        url: '/admin/teams/add',
        type: 'POST',
        contentType: 'text/plain; charset=UTF-8',
        data: $('.user-config-csv-upload textarea').val(),
        success: function(results) {
            console.log(results);
            $('.user-config-csv-upload')
                .append("<p>"+results+"</p>");
        },
        error: function(xhr, status, err) {

            console.log("FAILED");
            $('.user-config-csv-upload')
                .append("<p>"+xhr.responseText+"</p>");
        },
    });
};
