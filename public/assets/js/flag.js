$(document).ready(function() {
    getList();
});

$('#hiddenModal').on("click", "#flag-submit", function() {
    var flagValue = $('#flag-value').val();
    var challengeValue = $(this).data("challenge");

    if (flagValue.length > 0) {
        $.ajax({
            url: 'flag/CheckFlag',
            type: 'POST',
            dataType: 'html',
            data: { 
                challenge: challengeValue,
                flag: flagValue
            },
            success: function(value) {
                if (value == 'true') {
                    $('#flag-form').removeClass('has-error').addClass('has-success');
                } else {
                    $('#flag-form').removeClass('has-success').addClass('has-error');
                }
            },
        });
    }
});

function getList() {
    var url = 'flag/GetFlags'
    $.getJSON(url, function(json) {
        //console.dir(JSON.stringify(json, null, 2));
        makeList(json);
    });
}

function makeList(flags) {
    for (i = 0; i < flags.length; i++) {
        var list = $('<button/>').attr({
            type: 'button',
            class: 'btn btn-default btn-block',
            'data-toggle': 'modal',
            'data-target': '#currentFlag',
            onclick: 'makeModal("' + flags[i].name + '", "' + flags[i].challenge + '")'
        });
        list.append(flags[i].name);
        $("#flag-list").append(list);
    }
}
/*
 *  STILL NEED TO EDIT MODAL
 */
function makeModal(name, challenge) {
    // Clear it out first
    $("#hiddenModal").empty();

    var modal = $('<div/>').attr({
        class: "modal fade",
        id: "currentFlag",
        tabindex: "-1",
        role: "dialog",
        "aria-labelledby": "myModalLabel"
    });

    var content = $('<div/>').attr({
        class: "modal-content"
    });

    var header = $('<div/>').attr({
        class: "modal-header"
    });

    var hButton = $("<button type='button' class='close' data-dismiss='modal' aria-label='Close'><span aria-hidden='true'>&times;</span></button>");

    var title = $('<h4/>').attr({
        class: "modal-title",
    });

    var dialog = $('<div/>').attr({
        class: "modal-dialog",
    });

    var body = $('<div/>').attr({
        class: "modal-body"
    });

    var flagForm = $('<div/>').attr({
        id: "flag-form"
    });

    var flagLabel = $('<label/>').attr({
        class: "control-label",
        "for": "flag-value",
    });

    var flagInput = $('<input/>').attr({
        type: "text",
        class: "form-control",
        id: "flag-value",
        placeholder: "Flag{123}"
    });

    var footer = $('<div/>').attr({
        class: "modal-footer"
    });

    var closeButton = $('<button/>').attr({
        type: "button",
        class: "btn btn-default",
        "data-dismiss": "modal"
    });

    var submitButton = $('<button/>').attr({
        type: "submit",
        id: "flag-submit",
        class: "btn btn-primary",
        "data-challenge": challenge
    });

    // Append each tag into each other making sure they are in order as well
    closeButton.append("Close");
    footer.append(closeButton);
    submitButton.append("Submit");
    footer.append(submitButton);
    title.append(name);
    header.append(hButton);
    header.append(title);
    flagLabel.append("Enter Flag:");
    flagForm.append(flagLabel);
    flagForm.append(flagInput);
    body.append(flagForm);
    content.append(header);
    content.append(body);
    content.append(footer);
    dialog.append(content);
    modal.append(dialog);

    $("#hiddenModal").append(modal);

}
