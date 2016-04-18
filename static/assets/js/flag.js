$(document).ready(function() {
    getList();
});

$('#hiddenModal').on('click', '#flag-submit', submitFlag);
$('#hiddenModal').on('keypress', '#flag-value', function(e) {
    if (e.which == 13) {
        submitFlag();
        return false;
    }
});

function submitFlag() {
    var flagValue = $('#flag-value').val();
    var challengeValue = $('#flag-submit').data('challenge');

    if (flagValue.length > 0) {
        $.ajax({
            url: '/challenge/verify',
            type: 'POST',
            dataType: 'html',
            data: {
                flag: flagValue,
                challenge: challengeValue,
            },
            success: function(value) {
                if (value == '0') {
                    $('#flag-form').removeClass('has-error').addClass('has-success');
                    $('#flag-form .glyphicon').removeClass('glyphicon-remove').addClass('glyphicon-ok');
                } else if (value == '1') {
                    $('#flag-form').removeClass('has-success').addClass('has-error');
                    $('#flag-form .glyphicon').removeClass('glyphicon-ok').addClass('glyphicon-remove');
                } else if (value == '2') {
                    $('#flag-form').removeClass('has-success has-error').addClass('has-warning');
                    $('#flag-form .glyphicon').removeClass('glyphicon-ok glyphicon-remove').addClass('glyphicon-warning-sign');
                }
            },
        });
    }
}

$('input#flag-value').keypress(function(e) {
    if (e.which == 13) {
        $('#flag-submit').submit();
        return false;
    }
});

var challenges;

function getList() {
    var url = '/challenges'
    $.getJSON(url, function(json) {
        challenges = json;
        $('.page-header').append(' <small>' + challenges[0].group + '</small>');
        makeList(json);
    });
}

function makeList(chal) {
    for (i = 0; i < chal.length; i++) {
        var list = $('<button/>').attr({
            type: 'button',
            class: 'btn btn-default btn-block',
            'data-toggle': 'modal',
            'data-target': '#currentChallenge',
            onclick: 'makeModal(challenges[' + i + '])'
        });
        list.append(chal[i].name);
        //list.append('<span class="badge">' + chal[i].points + '</span>');
        $('#challenge-list').append(list);
    }
}

function makeModal(chal) {
    var titleName = escapeHtml(chal.name);
    var description = chal.points;
    $('#hiddenModal').empty();

    var modal = $('<div/>').attr({
        class: 'modal fade',
        id: 'currentChallenge',
        tabindex: '-1',
        role: 'dialog',
        'aria-labelledby': 'myModalLabel'
    });

    var content = $('<div/>').attr({
        class: 'modal-content'
    });

    var header = $('<div/>').attr({
        class: 'modal-header'
    });

    var hButton = $('<button type="button" class="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>');

    var title = $('<h4/>').attr({
        class: 'modal-title'
    });

    var dialog = $('<div/>').attr({
        class: 'modal-dialog'
    });

    var body = $('<div/>').attr({
        class: 'modal-body'
    });

    var challengeForm = $('<div/>').attr({
        id: 'flag-form',
        class: 'form-group has-feedback'
    });

    var flagLabel = $('<label/>').attr({
        class: 'control-label',
        'for': 'flag-value'
    });

    var flagInput = $('<input/>').attr({
        type: 'text',
        class: 'form-control',
        id: 'flag-value',
        placeholder: 'Flag{123}'
    });

    var footer = $('<div/>').attr({
        class: 'modal-footer'
    });

    var closeButton = $('<button/>').attr({
        type: 'button',
        class: 'btn btn-default',
        'data-dismiss': 'modal'
    });

    var submitButton = $('<button/>').attr({
        type: 'submit',
        id: 'flag-submit',
        class: 'btn btn-primary',
        'data-challenge': chal.name
    });

    // Append each tag into each other making sure they are in order as well
    closeButton.append('Close');
    footer.append(closeButton);
    submitButton.append('Submit');
    footer.append(submitButton);
    title.append(titleName);
    header.append(hButton);
    header.append(title);
    body.append('<h5>Points: ' + description + '</h5>');
    flagLabel.append('Enter Flag:');
    challengeForm.append(flagLabel);
    challengeForm.append(flagInput);
    challengeForm.append('<span class="glyphicon form-control-feedback" aria-hidden="true"></span><span id="inputSuccess2Status" class="sr-only"></span>');
    body.append(challengeForm);
    content.append(header);
    content.append(body);
    content.append(footer);
    dialog.append(content);
    modal.append(dialog);

    $('#hiddenModal').append(modal);
}

var entityMap = {
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': '&quot;',
    "'": '&#39;',
    "/": '&#x2F;'
};

function escapeHtml(string) {
    return String(string).replace(/[&<>"'\/]/g, function(s) {
        return entityMap[s];
    });
}
