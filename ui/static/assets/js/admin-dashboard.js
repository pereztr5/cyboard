/* Setup sub-sections -- Config and Score breakdown*/
$(function() {
    // Setup sub-dashboard selection
    showConfigPanels();
    refreshBreakdowns();

    // Event handlers to switch between configuration panels
    $('#select-config').on(  'change', showConfigPanels );
    $('#select-score-bd').on('change', showScoreBreakdownPanels );
});

function selectSubDash(section) {
    var sections = $(".top-level-dashboard").children("div");

    sections.not(section).hide();
    $(section).show();
}

function showConfigPanels() {
    selectSubDash( ".config-dash" );
}

function showScoreBreakdownPanels() {
    selectSubDash( ".score-bd-dash" );
}


/*
 *     User Configuration
 */
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
        if (orig === undefined && orig !== text && (text !== "" || cell.index() === column_keys.indexOf("adminof"))) {
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
                    url: '/api/admin/team/'+teamid,
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
                    url: '/api/admin/team/'+teamid,
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
        url: '/api/admin/teams',
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

    var user_cfg = $('.user-config-table');
    if (!user_cfg.length) return;
    $.get( "/api/admin/teams", function(teams) {
        // Wipe the current table, replace with new data
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

/*
 *   Challenge Configuration
 */
$(function() {
    populateChallengesTable();

    $('#bonus-points-form').submit(function(e) {
        e.preventDefault();
        submitBonusPoints($(this));
    });

    /*
     * CHALLENGE CONFIGURATION TABLE
     */
    const flag_cfg = $('.flag-config-table');
    flag_cfg.editableTableWidget({});

    // --- Editing CTF ---
    // Flag cells as changed, to provide feedback to the flag admin
    flag_cfg.find('tbody').on('change', 'td', function(evt) {
        const cell = $(this);
        if (!cell.attr('data-original')) {
            cell.closest('tr').addClass('changed');
            cell.attr('data-original', '_');
        }
    });

    // --- Update Challenge for CTF ---
    flag_cfg.find('tbody').on("click", ".flag-update", function() {

        const status_node = $('.edit-flag-controls').find('.status');
        const row = $(this).closest('tr');
        if (!row.hasClass('changed')) {
            status_node.text("Skipping unchanged row");
            return;
        }

        const [name, group, flag, points, description] = $.map( row.find('td'), (c) => c.textContent);
        const updateOp = createFlagFromStrings(name, group, flag, points, description);

        const flagName = row.attr('data-flag-name');
        BootstrapDialog.confirm(`Update flag '${flagName}'?`, function(yes) {
            if(!yes) { return }

            $.ajax({
                url: `/api/ctf/flags/${flagName}`,
                type: 'PUT',
                contentType: 'application/json; charset=UTF-8',
                data: JSON.stringify(updateOp),
                success: populateChallengesTable,
                complete: (xhr) => status_node.text(xhr.responseText || `Updated: ${updateOp['name']}`),
            });
        });
    });

    // --- Delete Challenge ---
    flag_cfg.find('tbody').on("click", ".flag-del", function() {
        const flagName = $(this).closest('tr').attr('data-flag-name');

        BootstrapDialog.confirm(`Delete flag '${flagName}'?`, function(yes) {
            if(!yes) { return }

            $.ajax({
                url: `/api/ctf/flags/${flagName}`,
                type: 'DELETE',
                complete: populateChallengesTable,
            });
        });
    });

    /* CSV Upload extras */
    const placeholderFlagCsv =
        "     Name,     Group,                     Flag, Points, Description\n" +
        " crypto-1,       CTF,         flag{s3cReT2no1},    100,            \n" +
        " crypto-2,       CTF,      flag{trU5tTh3CENt_},    150,            \n" +
        "   prog-1,       CTF,   flag{wh0suRdaddaddy?!},     50,            \n" +
        "  Stage 1,      Wifi,          easy wasn't it?,     25,            \n" +
        "  Stage 2,      Wifi,      Getting tougher now,     50,            \n" +
        "  Stage 3,      Wifi,         j0nc4n5b3st0Pp3D,     75,            \n";

    $('.flag-config-csv-upload').find('textarea')
        .val(placeholderFlagCsv);

    $('.flag-config-csv-upload form').on('submit', function(e) {
        e.preventDefault();
        submitFlagTextareaCsv();
    });
});

// --- Add Flags ---
// Convert the plain-text CSV into JSON suitable to post to the server
// Errors are shown next to the submit button
var submitFlagTextareaCsv = function() {
    const status_node = $('.flag-config-csv-upload').find('.status');
    const flagsCsv = $('.flag-config-csv-upload textarea').val();

    // let rows = csv.split(/\n/)
    // .map((row) => row.split(/,/, 5)
    //     .map((col) => col.trimLeft()))
    // .filter((row) => row.length === 5);

    let rows;
    try {
        rows = $.csv.toArrays(flagsCsv, {onParseValue: (e)=>e.trimLeft()});
    } catch (e) {
        status_node.text(e.message);
        throw e;
    }

    // Remove header if present
    if (rows[0][0] === "Name" && rows[0][1] === "Group") {
        rows.shift();
    }

    const flagsToAdd = rows.map((row) => createFlagFromStrings(...row));

    $.ajax({
        url: `/api/ctf/flags`,
        type: 'POST',
        contentType: 'application/json; charset=UTF-8',
        data: JSON.stringify(flagsToAdd),
        complete: (xhr) => {
            status_node.text(xhr.responseText || `Added ${flagsToAdd.length} flags!`);
            populateChallengesTable();
        },
    });
};

// --- Refresh the Challenge listing table completely ---
var populateChallengesTable = function() {
    var chal_cfg = $('.flag-config-table');
    if (!chal_cfg.length) return;
    $.get( "/api/ctf/flags", function(chals) {
        // Wipe the current table, replace with new data
        var tbody = chal_cfg.find('tbody');
        tbody.empty();
        $.each(chals, function(i, chal) {
            /* Build the inner DOM elements of the <tr> */
            tbody.append($("<tr/>").attr('data-flag-name', chal['name'])
                    .append($('<td/>').text(chal['name']))
                    .append($('<td/>').text(chal['group']))
                    .append($('<td/>').text(chal['flag']))
                    .append($('<td/>').text(chal['points']))
                    .append($('<td/>').text(chal['description']))
                    .append($('<th/>').addClass("controls")
                        .append($('<div/>').addClass('btn-group')
                            // .append(newIconButton("fa-clipboard", "btn-success ctf-clip"))
                                .append(newIconButton("fa-pencil", "btn-warning flag-update"))
                                .append(newIconButton("fa-trash", "btn-danger flag-del"))
                        )
                    )
            );
        });
        chal_cfg.editableTableWidget({});
    }, "json");
};

function createFlagFromStrings(name, group, flag, points, description) {
    return {
        name: name,
        group: group,
        flag: flag,
        points: window.parseInt(points, 10),
        description: description,
    }
}

function submitBonusPoints(form) {
    const serialize = (inputName, fn) => fn(form.find(`input[name=${inputName}]`).val());
    const data = {
        teams:   serialize('teams',   (val) => val.replace(/\s+/g, '').split(/,/)),
        points:  serialize('points',  (val) => window.parseInt(val, 10)),
        details: serialize('details', (val) => val),
    }

    const status_node = form.parentsUntil('.bonus-display').parent().find('.status');
    $.ajax({
        url: '/api/admin/grant_bonus',
        type: 'POST',
        contentType: 'application/json; charset=UTF-8',
        data: JSON.stringify(data),
        complete: (xhr) => status_node.text(xhr.responseText || `${data.points} points awarded`),
    });
}

/*
 *    CTF Flag Score breakdowns
 */

var initialCountdown = 60; // 1 minute between refreshes
(function countdown(remaining) {
    var timerNode = $('.countdown');
    if(remaining === 0) {
        refreshBreakdowns();
        remaining = initialCountdown;
        timerNode.text(remaining + "s . Just refreshed!");
    } else {
        timerNode.text(remaining + "s ...");
    }
    return setTimeout(function(){ countdown(remaining - 10); }, 10000);
})(initialCountdown);

var refreshBreakdowns = function () {
    getFlagsBySubmissionCount();
    getEachTeamsCapturedFlags();
};

var getFlagsBySubmissionCount = function() {
    $.get( "/api/ctf/stats/subs_per_flag", function(res) {
        var target = $('.most-submitted-flag');
        var tbody = target.find('tbody');
        tbody.empty();
        $.each(res, function(i, bd) {
            tbody.append($('<tr/>')
                    .append($('<td>').text(bd['name']))
                    .append($('<td>').text(bd['group']))
                    .append($('<td>').text(bd['submissions']))
            );
        })
    }, "json");
};

var getEachTeamsCapturedFlags = function() {
    $.get( "/api/ctf/stats/teams_flags", function(res) {
        var target = $('.teams-captured-flags');
        var tbody = target.find('tbody');
        tbody.empty();
        $.each(res, function(i, bd) {
            tbody.append($('<tr/>')
                .append($('<td>').text(bd['team']))
                .append($('<td>').text(bd['flags'].join(", ")).addClass('text-justify'))
            );
        })
    }, "json");
};
