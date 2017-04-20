// Make a new DOM element with no parents, with the given fa_icon and string of button classes
var newIconButton = function(fa_icon, button_cls) {
    fa_icon = fa_icon || "fa-exclamation";
    return $('<button/>').attr('type', 'button').addClass('btn '+button_cls)
        .append($('<i>').attr('aria-hidden', true).addClass('fa '+fa_icon));
};

// --- Refresh the Challenge listing table completely ---
var populateChallengesTable = function() {
    $.get( "/ctf/config", function(teams) {
        // Wipe the current table, replace with new data
        var chal_cfg = $('.flag-config-table');
        var tbody = chal_cfg.find('tbody');
        tbody.empty();
        $.each(teams, function(i, team) {
            /* Build the inner DOM elements of the <tr> */
            tbody.append($("<tr/>")
                .append($('<td/>').text(team['name']))
                .append($('<td/>').text(team['group']))
                .append($('<td/>').text(team['flag']))
                .append($('<td/>').text(team['points']))
                .append($('<td/>').text(team['description']))
                // .append($('<th/>').addClass("controls")
                //     .append($('<div/>').addClass('btn-group')
                //         // .append(newIconButton("fa-clipboard", "btn-success user-clip"))
                //             .append(newIconButton("fa-pencil", "btn-warning user-update"))
                //             .append(newIconButton("fa-trash", "btn-danger user-del"))
                //     )
                // )
            );
        });
        // chal_cfg.editableTableWidget({});
    }, "json");
};

$(
    populateChallengesTable()
);