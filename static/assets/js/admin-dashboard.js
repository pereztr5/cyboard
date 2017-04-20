$(function() {
    /*
     * USER CONFIGURATION TABLE
     */
    var user_cfg = $('.user-config-table');
    user_cfg.editableTableWidget({});

    // --- Editing Users ---
    // Hardcoded columns keys. Used in communication with server
    var column_keys = ['name', 'group', 'number', 'ip', 'adminof', 'password'];

    // For all the editable cells,
    // get either its current text content, or the original content
    var getCellOrig = function(cell) {
        return cell.attr('data-original') || cell.text();
    };

    // Store the original value as edits are happening
    user_cfg.find('tbody').on('change', 'td', function(evt) {
        var cell = $(this),
            orig = cell.attr('data-original'),
            text = cell.text();

        cell.closest('tr').addClass('changed');

        // Only change the class & data if the values really changed.
        if (orig === undefined && orig !== text && text !== "") {
            if (cell.index() === column_keys.indexOf('password')) {
                cell.removeClass('placeholder');
            }
            cell.attr('data-original', text);
        } else {
            cell.removeAttr('data-original')
        }
    });

    // --- Update Users ---
    user_cfg.find('tbody').on("click", ".user-update", function() {

        var status_node = $('.edit-user-controls').find('.status');

        var row = $(this).closest('tr');
        if (!row.hasClass('changed')) {
            status_node.text("Skipping unchanged row");
            return;
        }

        var teamid = getCellOrig(row.find('td:nth-child(1)'));

        // Create object with necessary data for updates
        // Type checking is done server side
        // Result Eg.:
        //     {"password": "bad_example_dude", "number": "2"}
        var updateOp = {};

        $.each(row.find('td'), function(idx, html_cell) {
            var cell = $(html_cell);
            if (cell.attr('data-original') !== undefined) {
                var text = cell.text();
                if (cell.index() === column_keys.indexOf("number")) {
                    var num = parseInt(text, 10);
                    if (isNaN(num)) {
                        var err = "Not an integer (column #"+(idx+1)+"): "+text;
                        status_node.text(err);
                        throw err;
                    }
                    updateOp[column_keys[idx]] = num;
                } else {
                    updateOp[column_keys[idx]] = text;
                }
            }
        });

        BootstrapDialog.confirm("Update team '"+teamid+"'?", function(yes) {
            if(yes) {
                // TODO: Build out server side update route
                $.ajax({
                    url: '/admin/team/update/'+teamid,
                    type: 'PUT',
                    contentType: 'application/json; charset=UTF-8',
                    data: JSON.stringify(updateOp),
                    success: function(results) {
                        status_node.text("Update complete.");
                        populateUsersTable()
                    },
                    error: function(xhr, status, err) {
                        status_node.text(xhr.responseText);
                    },
                });
            }
        });
    });

    // --- Delete User ---
    //Get user of clicked row, confirm delete via dialog, call api, refresh
    user_cfg.find('tbody').on("click", ".user-del", function() {
        var teamNameNode = $(this).closest('tr').children('td:first');
        var teamid = getCellOrig(teamNameNode);
        BootstrapDialog.confirm("Delete team '"+teamid+"'?", function(yes) {
            if(yes) {
                $.ajax({
                    url: '/admin/team/delete/'+teamid,
                    type: 'DELETE',
                });
                populateUsersTable();
            }
        });
    });

    /* Copy clicked row to clipboard */
    user_cfg.find('tbody').on("click", ".user-clip", function() {
        /* oh.
         * This is really hard to do:
         *   http://davidzchen.com/tech/2016/01/19/bootstrap-copy-to-clipboard.html
         *   https://developers.google.com/web/updates/2015/04/cut-and-copy-commands?hl=en
         *
         * Uh, maybe later.
         */
    });

    /* CSV Upload extras */
    var placeholderUsersCsv =
        "     Name,     Group, Number,            IP, AdminOf, Password\n" +
        "    team1,  blueteam,      1,  192.168.90.1,        , smokestack\n" +
        "    team2,  blueteam,      2,  192.168.90.2,        , bologna\n" +
        "    team3,  blueteam,      3,  192.168.90.3,        , xmas_monkEE\n" +
        "    team4,  blueteam,      4,  192.168.90.4,        , d2hhdCB0aGUgZnVjayBkaWQgeW91IGV4cGVjdAo=\n" +
        " evilcorp,   redteam,    100, 192.168.199.1,     AIS, \"rUsstED\"\"CuR^mug3Ons\"\n" +
        "   himike, whiteteam,    255,       1.1.1.1,     CTF, challenge--m4ster\n" +
        " johnWifi, whiteteam,    256,       0.0.0.0,    Wifi, whatsAfirewall?\n" +
        "lugerJose, blackteam,    666,     6.6.6.255,        , \"!*,B4NGb4ng,*!\"\n";

    $('.user-config-csv-upload').find('textarea')
        .val(placeholderUsersCsv);

    $('.user-config-csv-upload form').on('submit', function(e) {
        e.preventDefault();
        submitTeamsTextareaCsv();
    });
});

// --- Add Users ---
// Post the plaintext CSV to the server, attempting to add all new users
// Errors are shown next to the submit button
var submitTeamsTextareaCsv = function() {
    var status_node = $('.user-config-csv-upload').find('.status');
    $.ajax({
        url: '/admin/teams/add',
        type: 'POST',
        contentType: 'text/csv; charset=UTF-8',
        data: $('.user-config-csv-upload textarea').val(),
        success: function(results) {
            status_node.text("OK.");
        },
        error: function(xhr, status, err) {
            status_node.text(xhr.responseText);
        },
        complete: function() {
            populateUsersTable();
        }
    });
};


// Make a new DOM element with no parents, with the given fa_icon and string of button classes
var newIconButton = function(fa_icon, button_cls) {
    fa_icon = fa_icon || "fa-exclamation";
    return $('<button/>').attr('type', 'button').addClass('btn '+button_cls)
        .append($('<i>').attr('aria-hidden', true).addClass('fa '+fa_icon));
};

// --- Refresh the Users table completely ---
var populateUsersTable = function() {

    $.get( "/admin/teams", function(teams) {
        // Wipe the current table, replace with new data
        var user_cfg = $('.user-config-table');
        var tbody = user_cfg.find('tbody');
        tbody.empty();
        $.each(teams, function(i, team) {
            /* Build the inner DOM elements of the <tr> */
            tbody.append($("<tr/>")
                .append($('<td/>').text(team['name']))
                .append($('<td/>').text(team['group']))
                .append($('<td/>').text(team['number']))
                .append($('<td/>').text(team['ip']))
                .append($('<td/>').text(team['adminof']))
                .append($('<td/>').text("{Unchanged}").addClass("placeholder"))
                .append($('<th/>').addClass("controls")
                    .append($('<div/>').addClass('btn-group')
                        // .append(newIconButton("fa-clipboard", "btn-success user-clip"))
                        .append(newIconButton("fa-pencil", "btn-warning user-update"))
                        .append(newIconButton("fa-trash", "btn-danger user-del"))
                    )
                )
            );
        });
        user_cfg.editableTableWidget({});
    }, "json");
};
