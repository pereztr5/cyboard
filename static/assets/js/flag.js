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
            url: '/flags/verify',
            type: 'POST',
            dataType: 'html',
            data: {
                challenge: challengeValue,
                flag: flagValue
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

function getList() {
    var url = '/flags'
    $.getJSON(url, function(json) {
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
            onclick: 'makeModal("' + flags[i].flagname + '", "' + flags[i].challenge + '")'
        });
        list.append(flags[i].flagname);
        $('#flag-list').append(list);
    }
}

function makeModal(name, challenge) {
    $('#hiddenModal').empty();

    var modal = $('<div/>').attr({
        class: 'modal fade',
        id: 'currentFlag',
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
        class: 'modal-title',
    });

    var dialog = $('<div/>').attr({
        class: 'modal-dialog',
    });

    var body = $('<div/>').attr({
        class: 'modal-body'
    });

    var flagForm = $('<div/>').attr({
        class: 'form-group has-feedback',
        id: 'flag-form',
    });

    var flagLabel = $('<label/>').attr({
        class: 'control-label',
        'for': 'flag-value',
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
        'data-challenge': challenge
    });

    // Append each tag into each other making sure they are in order as well
    closeButton.append('Close');
    footer.append(closeButton);
    submitButton.append('Submit');
    footer.append(submitButton);
    title.append(name);
    header.append(hButton);
    header.append(title);
    flagLabel.append('Enter Flag:');
    flagForm.append(flagLabel);
    flagForm.append(flagInput);
    flagForm.append('<span class="glyphicon form-control-feedback" aria-hidden="true"></span><span id="inputSuccess2Status" class="sr-only"></span>');
    body.append(flagForm);
    content.append(header);
    content.append(body);
    content.append(footer);
    dialog.append(content);
    modal.append(dialog);

    $('#hiddenModal').append(modal);

}
